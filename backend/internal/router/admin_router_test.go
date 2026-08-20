package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"intellisearch/internal/config"
	"intellisearch/internal/handlers"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"intellisearch/internal/services"
)

type allowingLimiter struct{}
type fakeRunner struct{}

func (allowingLimiter) Allow(context.Context, string, string, int, time.Duration) (bool, error) {
	return true, nil
}
func (fakeRunner) Answer(ctx context.Context, input services.AskInput) (services.AskResult, error) {
	return services.AskResult{Answer: "ok"}, nil
}
func (fakeRunner) SuggestFollowUps(ctx context.Context, userID *uuid.UUID, sessionID uuid.UUID) ([]string, error) {
	return nil, nil
}

func adminTestMux(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:admin-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&entities.User{}, &entities.SiteSettings{}, &entities.AIQueueConfig{}, &entities.AIProvider{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}, &entities.ImageResult{}, &entities.UsageLog{}, &entities.CrawlJob{}, &entities.SearchHistory{}, &entities.AnonymousUsage{}, &entities.Note{}, &entities.MapPoint{}, &entities.RegisterVisit{}); err != nil {
		t.Fatal(err)
	}
	seed := func(user *entities.User, password string) {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		user.PasswordHash = string(hash)
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	seed(&entities.User{ID: uuid.New(), Name: "Owner", Email: "owner@example.com", Role: entities.RoleSuperOwner, Status: entities.StatusActive}, "owner-pass")
	seed(&entities.User{ID: uuid.New(), Name: "Jane", Email: "jane@example.com", Role: entities.RoleGeneralUser, Status: entities.StatusActive}, "jane-pass")
	if err := db.Create(&entities.SiteSettings{ID: 1, SiteName: "Intellisearch"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10}).Error; err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{JWTSecret: "admin-test-secret-32-chars-minimum", JWTTTLHours: 24}
	authService := services.NewAuthService(repositories.NewUserRepository(db), repositories.NewQueueConfigRepository(db), cfg)
	userService := services.NewUserService(repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), t.TempDir())
	historyService := services.NewSearchHistoryService(repositories.NewSearchHistoryRepository(db), repositories.NewMessageRepository(db), services.NewLLMService(repositories.NewProviderRepository(db), cfg.JWTSecret), repositories.NewQueueConfigRepository(db))
	adminService := services.NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), cfg.JWTSecret, t.TempDir())
	statsService := services.NewStatsService(repositories.NewUsageLogRepository(db), repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil, repositories.NewAnonymousUsageRepository(db), repositories.NewRegisterVisitRepository(db))
	aiHandler := handlers.NewAIHandler(fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), allowingLimiter{}, authService)
	t.Cleanup(aiHandler.Stop)
	adminHandler := handlers.NewAdminHandler(userService, adminService, statsService, services.NewOllamaService())
	appsHandler := handlers.NewAppsHandler(services.NewNoteService(repositories.NewNoteRepository(db)), services.NewTranslateService(""), allowingLimiter{})
	pollinationsHandler := handlers.NewPollinationsHandler(adminService, services.NewPollinationsService("https://media.pollinations.ai"))
	siteService := services.NewSiteService(repositories.NewSiteRepository(db))
	visitorHandler := handlers.NewVisitorHandler(repositories.NewRegisterVisitRepository(db))
	mux := New("*", t.TempDir(), siteService, handlers.NewAuthHandler(authService), handlers.NewUserHandler(userService, historyService), nil, aiHandler, adminHandler, appsHandler, pollinationsHandler, visitorHandler, authService)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, db
}

func loginToken(t *testing.T, server *httptest.Server, email, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	response, err := http.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login failed (%d): %s", response.StatusCode, payload)
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data.Token
}

func call(t *testing.T, server *httptest.Server, method, path, token string, body []byte) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, server.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, payload
}

func TestRegisterVisitTrackingEndToEnd(t *testing.T) {
	server, _ := adminTestMux(t)

	// Public tracking: an anonymous POST (no identity) records a new visit.
	status, payload := call(t, server, http.MethodPost, "/api/v1/stats/register-visit", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"new":true`)) {
		t.Fatalf("expected the first register visit to be recorded as new: %d %s", status, payload)
	}

	// A token-identified visitor is counted once: the first call is new, the
	// replay (same X-Visitor-ID) must be a no-op.
	visitor := uuid.NewString()
	visit := func() (int, []byte) {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/stats/register-visit", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("X-Visitor-ID", visitor)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, body
	}
	status, payload = visit()
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"new":true`)) {
		t.Fatalf("new token should record a new visit: %d %s", status, payload)
	}
	status, payload = visit()
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"new":false`)) {
		t.Fatalf("replay must not count as a new visit: %d %s", status, payload)
	}

	// The super owner sees the unique-visitor summary (two register-page
	// visitors: the cookie-less one and the header-token one).
	ownerToken := loginToken(t, server, "owner@example.com", "owner-pass")
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/stats/visitors", ownerToken, nil)
	if status != http.StatusOK {
		t.Fatalf("visitors endpoint failed: %d %s", status, payload)
	}
	var summary struct {
		Data struct {
			RegisteredUsers struct {
				Total int64 `json:"total"`
			} `json:"registeredUsers"`
			AnonymousVisitors struct {
				Total int64 `json:"total"`
			} `json:"anonymousVisitors"`
			RegisterPageVisits struct {
				Total int64 `json:"total"`
				Daily []struct {
					Count int64 `json:"count"`
				} `json:"daily"`
			} `json:"registerPageVisits"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Data.RegisterPageVisits.Total != 2 || len(summary.Data.RegisterPageVisits.Daily) != 7 {
		t.Fatalf("expected 2 unique register-page visitors and 7 daily points, got total=%d days=%d (%s)", summary.Data.RegisterPageVisits.Total, len(summary.Data.RegisterPageVisits.Daily), payload)
	}
	if summary.Data.RegisterPageVisits.Daily[6].Count != 2 {
		t.Fatalf("today's register-visit bucket should count both visitors, got %d", summary.Data.RegisterPageVisits.Daily[6].Count)
	}
	// The seeded super owner + jane count as registered users.
	if summary.Data.RegisteredUsers.Total != 2 {
		t.Fatalf("expected exactly the two seeded accounts as registered users, got %d", summary.Data.RegisteredUsers.Total)
	}

	// General users are forbidden from the visitor summary.
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")
	status, _ = call(t, server, http.MethodGet, "/api/v1/admin/stats/visitors", janeToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for general user, got %d", status)
	}
}

func TestAdminRequiresSuperOwner(t *testing.T) {
	server, _ := adminTestMux(t)
	ownerToken := loginToken(t, server, "owner@example.com", "owner-pass")
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")

	// General user is forbidden on every admin route.
	status, _ := call(t, server, http.MethodGet, "/api/v1/admin/users", janeToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for general user, got %d", status)
	}
	// Anonymous is rejected (401) before the role check.
	status, _ = call(t, server, http.MethodGet, "/api/v1/admin/stats", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for anonymous, got %d", status)
	}
	// Super owner reaches the endpoint and sees the seeded users.
	status, payload := call(t, server, http.MethodGet, "/api/v1/admin/users", ownerToken, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for super owner, got %d: %s", status, payload)
	}
	var envelope struct {
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Data.Total < 2 {
		t.Fatalf("expected at least 2 users, got %s (err=%v)", payload, err)
	}
}

func TestAdminSearchAndDeleteUser(t *testing.T) {
	server, _ := adminTestMux(t)
	token := loginToken(t, server, "owner@example.com", "owner-pass")
	status, payload := call(t, server, http.MethodPost, "/api/v1/admin/users", token, []byte(`{"name":"New User","email":"new@example.com","password":"pass123","role":"general_user","aiDailyQuota":8}`))
	if status != http.StatusOK {
		t.Fatalf("create user failed: %d %s", status, payload)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &created); err != nil || created.Data.ID == "" {
		t.Fatalf("created user missing id: %s", payload)
	}
	// Search finds it.
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/users?q=new", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte("new@example.com")) {
		t.Fatalf("search users failed: %d %s", status, payload)
	}
	// Delete removes it.
	status, payload = call(t, server, http.MethodDelete, "/api/v1/admin/users/"+created.Data.ID, token, nil)
	if status != http.StatusOK {
		t.Fatalf("delete user failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/users?q=new", token, nil)
	if status != http.StatusOK || bytes.Contains(payload, []byte("new@example.com")) {
		t.Fatalf("deleted user still present: %d %s", status, payload)
	}
}

func TestAskAnonymousLimitEndToEnd(t *testing.T) {
	server, _ := adminTestMux(t)
	// First anonymous ask from this IP: allowed and issues a visitor token.
	status, payload := call(t, server, http.MethodPost, "/api/v1/ask", "", []byte(`{"query":"first question"}`))
	if status != http.StatusOK {
		t.Fatalf("expected first anonymous ask to succeed, got %d: %s", status, payload)
	}
	var envelope struct {
		Data struct {
			VisitorID string `json:"visitorId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Data.VisitorID == "" {
		t.Fatalf("expected a visitorId to be issued, got %s (err=%v)", payload, err)
	}
	// Second anonymous ask (token cleared): blocked by the per-IP claim.
	status, payload = call(t, server, http.MethodPost, "/api/v1/ask", "", []byte(`{"query":"second question"}`))
	if status != http.StatusTooManyRequests || !bytes.Contains(payload, []byte("AISY02004")) {
		t.Fatalf("expected 429 AISY02004 for reused IP, got %d: %s", status, payload)
	}
	// A signed-in user from the same IP is exempt from the anonymous limit.
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")
	status, payload = call(t, server, http.MethodPost, "/api/v1/ask", janeToken, []byte(`{"query":"signed in question"}`))
	if status != http.StatusOK {
		t.Fatalf("signed-in users must be exempt from the anonymous limit, got %d: %s", status, payload)
	}
}

func TestAdminLogoDeleteAndTrends(t *testing.T) {
	server, db := adminTestMux(t)
	token := loginToken(t, server, "owner@example.com", "owner-pass")

	// Upload a logo, then delete it so branding falls back to the default.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("\x89PNG\r\n\x1a\nfakelogo")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/admin/site-settings/logo", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("logo upload failed: %d %s", response.StatusCode, payload)
	}
	var uploaded struct {
		Data struct {
			LogoURL string `json:"logoUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &uploaded); err != nil || uploaded.Data.LogoURL == "" {
		t.Fatalf("uploaded logo missing url: %s", payload)
	}
	var settings entities.SiteSettings
	if err := db.First(&settings, 1).Error; err != nil || settings.LogoURL == nil {
		t.Fatalf("logo not persisted: %v (err=%v)", settings.LogoURL, err)
	}

	status, payload := call(t, server, http.MethodDelete, "/api/v1/admin/site-settings/logo", token, nil)
	if status != http.StatusOK {
		t.Fatalf("delete logo failed: %d %s", status, payload)
	}
	if err := db.First(&settings, 1).Error; err != nil || settings.LogoURL != nil {
		t.Fatalf("logo not cleared: %v (err=%v)", settings.LogoURL, err)
	}
	// Public site endpoint reflects the fallback (null logoUrl).
	status, payload = call(t, server, http.MethodGet, "/api/v1/site", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"logoUrl":null`)) {
		t.Fatalf("public site did not fall back to default logo: %d %s", status, payload)
	}

	// Trends endpoint returns 7 daily and 8 weekly points (all zeros without data).
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/stats/trends", token, nil)
	if status != http.StatusOK {
		t.Fatalf("trends failed: %d %s", status, payload)
	}
	var trends struct {
		Data struct {
			Daily []struct {
				Count int64 `json:"count"`
			} `json:"daily"`
			Weekly []struct {
				Count int64 `json:"count"`
			} `json:"weekly"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &trends); err != nil {
		t.Fatal(err)
	}
	if len(trends.Data.Daily) != 7 || len(trends.Data.Weekly) != 8 {
		t.Fatalf("expected 7 daily and 8 weekly points, got %d and %d", len(trends.Data.Daily), len(trends.Data.Weekly))
	}

	// Insert a usage log today and confirm it shows up in today's daily point.
	now := time.Now().UTC()
	usageRepo := repositories.NewUsageLogRepository(db)
	uid := uuid.New()
	if err := usageRepo.Create(&entities.UsageLog{UserID: &uid, Query: "sample", Status: "completed", LatencyMS: 10, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/stats/trends", token, nil)
	if status != http.StatusOK {
		t.Fatalf("trends after insert failed: %d", status)
	}
	var withData struct {
		Data struct {
			Daily []struct {
				Label string `json:"label"`
				Count int64  `json:"count"`
			} `json:"daily"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &withData); err != nil {
		t.Fatal(err)
	}
	today := now.Format("2006-01-02")
	if withData.Data.Daily[6].Label != today || withData.Data.Daily[6].Count != 1 {
		t.Fatalf("expected today's point (%s) to count 1, got label=%s count=%d", today, withData.Data.Daily[6].Label, withData.Data.Daily[6].Count)
	}

	// Admin stats never leak verbatim queries: topQueries are masked.
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/stats", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"topQueries"`)) || bytes.Contains(payload, []byte(`"query":"sample"`)) {
		t.Fatalf("admin stats must mask queries: %d %s", status, payload)
	}

	// The trending-words endpoint aggregates the query into a term without
	// revealing it, for both windows.
	for _, window := range []string{"daily", "weekly"} {
		status, payload = call(t, server, http.MethodGet, "/api/v1/admin/stats/trending-words?window="+window, token, nil)
		if status != http.StatusOK {
			t.Fatalf("trending-words (%s) failed: %d %s", window, status, payload)
		}
		var words struct {
			Data struct {
				Window  string `json:"window"`
				Buckets []struct {
					Top []struct {
						Word  string `json:"word"`
						Count int64  `json:"count"`
					} `json:"top"`
				} `json:"buckets"`
				Overall []struct {
					Word string `json:"word"`
				} `json:"overall"`
			} `json:"data"`
		}
		if err := json.Unmarshal(payload, &words); err != nil {
			t.Fatal(err)
		}
		expected := 7
		if window == "weekly" {
			expected = 8
		}
		if words.Data.Window != window || len(words.Data.Buckets) != expected {
			t.Fatalf("trending-words (%s): unexpected window/buckets %s/%d", window, words.Data.Window, len(words.Data.Buckets))
		}
		// "sample" is a 6-letter content word, so the tokenizer keeps it — it must
		// surface in the overall top terms (the aggregated form, not a raw query).
		found := false
		for _, term := range words.Data.Overall {
			if term.Word == "sample" {
				found = true
			}
		}
		if !found {
			t.Fatalf("trending-words (%s) should contain the term \"sample\", got %s", window, payload)
		}
	}
}

func TestAdminProviderQueueAndBrandingLive(t *testing.T) {
	server, db := adminTestMux(t)
	token := loginToken(t, server, "owner@example.com", "owner-pass")

	// A provider's API key must be encrypted at rest.
	status, payload := call(t, server, http.MethodPost, "/api/v1/admin/ai/providers", token, []byte(`{"name":"cloud","providerType":"openai_compatible","baseUrl":"https://api.openai.com","model":"gpt-4o-mini","apiKey":"sk-secret","isActive":true}`))
	if status != http.StatusOK {
		t.Fatalf("create provider failed: %d %s", status, payload)
	}
	var provider struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &provider); err != nil || provider.Data.ID == "" {
		t.Fatal("provider not created", payload)
	}
	var stored entities.AIProvider
	if err := db.First(&stored, "id = ?", provider.Data.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.APIKeyEncrypted == nil || *stored.APIKeyEncrypted == "" || bytes.Contains([]byte(*stored.APIKeyEncrypted), []byte("sk-secret")) {
		t.Fatalf("api key not encrypted at rest: %v", stored.APIKeyEncrypted)
	}

	// Queue config updates persist and are reflected by the runtime.
	status, payload = call(t, server, http.MethodPatch, "/api/v1/admin/ai/queue-config", token, []byte(`{"maxConcurrent":6,"maxQueueSize":40,"requestTimeoutMs":120000,"perUserRateLimit":5,"sessionTtlHours":72}`))
	if status != http.StatusOK {
		t.Fatalf("update queue config failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/ai/queue-config", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"maxConcurrent":6`)) || !bytes.Contains(payload, []byte(`"sessionTtlHours":72`)) {
		t.Fatalf("queue config not persisted: %d %s", status, payload)
	}

	// Branding updates immediately reflect on the public site endpoint.
	status, _ = call(t, server, http.MethodPatch, "/api/v1/admin/site-settings", token, []byte(`{"siteName":"Acme Search","tagline":"Research distilled"}`))
	if status != http.StatusOK {
		t.Fatalf("update site settings failed: %d", status)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/site", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte("Acme Search")) {
		t.Fatalf("public site did not reflect branding: %d %s", status, payload)
	}

	// Statistics respond for the owner.
	status, _ = call(t, server, http.MethodGet, "/api/v1/admin/stats", token, nil)
	if status != http.StatusOK {
		t.Fatalf("user stats failed: %d", status)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/stats/ai", token, nil)
	if status != http.StatusOK {
		t.Fatalf("ai stats failed: %d", status)
	}
	// Token usage cards are part of the AI-stats response.
	if !bytes.Contains(payload, []byte(`"totalInputTokens"`)) || !bytes.Contains(payload, []byte(`"totalOutputTokens"`)) || !bytes.Contains(payload, []byte(`"tokensPerSec"`)) {
		t.Fatalf("ai stats missing token fields: %s", payload)
	}
}

func TestMeReturnsUsage(t *testing.T) {
	server, _ := adminTestMux(t)
	token := loginToken(t, server, "jane@example.com", "jane-pass")
	status, payload := call(t, server, http.MethodGet, "/api/v1/me", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"role":"general_user"`)) {
		t.Fatalf("me endpoint failed: %d %s", status, payload)
	}
}

func TestSearchHistoryEndpoints(t *testing.T) {
	server, db := adminTestMux(t)
	token := loginToken(t, server, "jane@example.com", "jane-pass")

	// Anonymous is rejected (401) on every history route.
	for _, path := range []string{"/api/v1/me/history", "/api/v1/me/history/suggestions"} {
		status, _ := call(t, server, http.MethodGet, path, "", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401 for anonymous on %s, got %d", path, status)
		}
	}

	// Seed a few history entries for jane, then list them newest first.
	historyRepo := repositories.NewSearchHistoryRepository(db)
	uid := uuid.New()
	var jane entities.User
	if err := db.First(&jane, "email = ?", "jane@example.com").Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i, q := range []string{"cheapest flights", "best ramen in tokyo", "cheapest flights"} {
		if err := historyRepo.Create(&entities.SearchHistory{ID: uint64(now.UnixNano()) + uint64(i), UserID: jane.ID, Query: q, CreatedAt: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	// An entry for another user must not leak into jane's history.
	if err := historyRepo.Create(&entities.SearchHistory{ID: uint64(now.UnixNano()) + 100, UserID: uid, Query: "private query", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	status, payload := call(t, server, http.MethodGet, "/api/v1/me/history", token, nil)
	if status != http.StatusOK {
		t.Fatalf("history list failed: %d %s", status, payload)
	}
	var listed struct {
		Data struct {
			Items []struct {
				Query string `json:"query"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data.Items) != 3 || listed.Data.Items[0].Query != "cheapest flights" {
		t.Fatalf("unexpected history listing: %s", payload)
	}

	// Suggestions degrade gracefully to an empty list (no provider configured
	// in the test mux), never an error.
	status, payload = call(t, server, http.MethodGet, "/api/v1/me/history/suggestions", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"suggestions":[]`)) {
		t.Fatalf("suggestions should degrade to empty list: %d %s", status, payload)
	}

	// Paginated mode: page=1&page_size=2 returns the 2 most recent items + total.
	status, payload = call(t, server, http.MethodGet, "/api/v1/me/history?page=1&page_size=2", token, nil)
	if status != http.StatusOK {
		t.Fatalf("paginated history failed: %d %s", status, payload)
	}
	var paginated struct {
		Data struct {
			Items []struct {
				Query string `json:"query"`
			} `json:"items"`
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"pageSize"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &paginated); err != nil {
		t.Fatal(err)
	}
	if paginated.Data.Total != 3 || paginated.Data.Page != 1 || paginated.Data.PageSize != 2 {
		t.Fatalf("unexpected pagination metadata: total=%d page=%d pageSize=%d (%s)", paginated.Data.Total, paginated.Data.Page, paginated.Data.PageSize, payload)
	}
	if len(paginated.Data.Items) != 2 || paginated.Data.Items[0].Query != "cheapest flights" {
		t.Fatalf("unexpected page 1 items: %s", payload)
	}
	// Page 2 returns the remaining item.
	status, payload = call(t, server, http.MethodGet, "/api/v1/me/history?page=2&page_size=2", token, nil)
	if status != http.StatusOK {
		t.Fatalf("page 2 failed: %d %s", status, payload)
	}
	if err := json.Unmarshal(payload, &paginated); err != nil {
		t.Fatal(err)
	}
	if paginated.Data.Total != 3 || len(paginated.Data.Items) != 1 {
		t.Fatalf("unexpected page 2: total=%d items=%d (%s)", paginated.Data.Total, len(paginated.Data.Items), payload)
	}

	// Clearing wipes jane's history only.
	status, payload = call(t, server, http.MethodDelete, "/api/v1/me/history", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"cleared":true`)) {
		t.Fatalf("clear history failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/me/history", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"items":[]`)) {
		t.Fatalf("history not cleared: %d %s", status, payload)
	}
	var remaining int64
	if err := db.Model(&entities.SearchHistory{}).Count(&remaining).Error; err != nil || remaining != 1 {
		t.Fatalf("expected only the other user's entry to remain, got %d (err=%v)", remaining, err)
	}
}

func TestRegisterEndpoint(t *testing.T) {
	server, _ := adminTestMux(t)

	// Valid registration returns a token and the new profile.
	status, payload := call(t, server, http.MethodPost, "/api/v1/auth/register", "", []byte(`{"name":"Registered","email":"registered@example.com","password":"password-123"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"token"`)) || !bytes.Contains(payload, []byte(`"role":"general_user"`)) {
		t.Fatalf("register failed: %d %s", status, payload)
	}
	// The new account can sign in.
	if token := loginToken(t, server, "registered@example.com", "password-123"); token == "" {
		t.Fatal("registered account could not sign in")
	}
	// Duplicate email is rejected with AUTH01005.
	status, payload = call(t, server, http.MethodPost, "/api/v1/auth/register", "", []byte(`{"name":"Jane Again","email":"jane@example.com","password":"password-123"}`))
	if status != http.StatusConflict || !bytes.Contains(payload, []byte(`"errorCode":"AUTH01005"`)) {
		t.Fatalf("duplicate register: expected 409 AUTH01005, got %d %s", status, payload)
	}
	// Weak password is rejected with AUTH01004.
	status, payload = call(t, server, http.MethodPost, "/api/v1/auth/register", "", []byte(`{"name":"Weak","email":"weak@example.com","password":"short"}`))
	if status != http.StatusBadRequest || !bytes.Contains(payload, []byte(`"errorCode":"AUTH01004"`)) {
		t.Fatalf("weak register: expected 400 AUTH01004, got %d %s", status, payload)
	}
}

func TestAdminOllamaEndpoints(t *testing.T) {
	server, _ := adminTestMux(t)
	ownerToken := loginToken(t, server, "owner@example.com", "owner-pass")
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")

	// Fake Ollama server answering the introspection endpoints.
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest","size":2019393183,"details":{"parameter_size":"3.2B","quantization_level":"Q4_K_M"}}]}`))
		case "/api/version":
			_, _ = w.Write([]byte(`{"version":"0.5.7"}`))
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest","cpu":"99%","gpu":"0%","memory":"1.6GB/3.8GB"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	// General users are forbidden from introspecting the server.
	status, _ := call(t, server, http.MethodGet, "/api/v1/admin/ai/ollama/models?baseUrl="+ollama.URL, janeToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for general user, got %d", status)
	}

	// Super owner lists models.
	status, payload := call(t, server, http.MethodGet, "/api/v1/admin/ai/ollama/models?baseUrl="+ollama.URL, ownerToken, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte("llama3.2:latest")) {
		t.Fatalf("ollama models failed: %d %s", status, payload)
	}
	// Health includes version and running-model stats.
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/ai/ollama/health?baseUrl="+ollama.URL, ownerToken, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"version":"0.5.7"`)) || !bytes.Contains(payload, []byte(`"cpu":"99%"`)) {
		t.Fatalf("ollama health failed: %d %s", status, payload)
	}
	// An invalid base URL is a 400 with ADMN06001.
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/ai/ollama/models?baseUrl=ftp%3A%2F%2Fnope", ownerToken, nil)
	if status != http.StatusBadRequest || !bytes.Contains(payload, []byte(`"errorCode":"ADMN06001"`)) {
		t.Fatalf("invalid ollama url: expected 400 ADMN06001, got %d %s", status, payload)
	}
}

func TestAdminPollinationsEndpoints(t *testing.T) {
	server, _ := adminTestMux(t)
	ownerToken := loginToken(t, server, "owner@example.com", "owner-pass")
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")

	// Fake Pollinations account + media API.
	poll := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/account/balance":
			// The real endpoint returns a bare JSON number (JSON-quoted string
			// when format=json is used); never an object.
			_, _ = w.Write([]byte(`9.5`))
		case "/account/profile":
			_, _ = w.Write([]byte(`{"githubUsername":"dev","image":null,"communityEndpointsAllowed":false}`))
		case "/account/key":
			_, _ = w.Write([]byte(`{"valid":true,"type":"secret","permissions":{"account":["usage"]},"rateLimitEnabled":false}`))
		case "/account/usage":
			_, _ = w.Write([]byte(`{"usage":[{"timestamp":"2026-08-17 10:00:00","model":"openai","cost_usd":0.001}],"count":1}`))
		case "/account/usage/daily":
			_, _ = w.Write([]byte(`{"usage":[{"date":"2026-08-17","requests":3,"cost_usd":0.003}],"count":1}`))
		case "/v1/models":
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"openai","object":"model"},{"id":"flux","object":"model"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer poll.Close()

	// Save a Pollinations provider pointing at the fake account API.
	status, payload := call(t, server, http.MethodPost, "/api/v1/admin/ai/providers", ownerToken, []byte(`{"name":"polli","providerType":"pollinations","baseUrl":"`+poll.URL+`","model":"openai","apiKey":"sk-polli","isActive":true}`))
	if status != http.StatusOK {
		t.Fatalf("create pollinations provider failed: %d %s", status, payload)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &created); err != nil || created.Data.ID == "" {
		t.Fatal("provider missing id", payload)
	}

	// General users are forbidden.
	status, _ = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/account", janeToken, []byte(`{"providerId":"`+created.Data.ID+`"}`))
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for general user, got %d", status)
	}

	// Super owner reads account (balance + profile + key) via providerId — the
	// stored API key is decrypted server-side, never sent to the browser.
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/account", ownerToken, []byte(`{"providerId":"`+created.Data.ID+`"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"balance":9.5`)) || !bytes.Contains(payload, []byte(`"githubUsername":"dev"`)) || !bytes.Contains(payload, []byte(`"valid":true`)) {
		t.Fatalf("pollinations account failed: %d %s", status, payload)
	}
	// Usage + daily usage.
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/usage?days=7", ownerToken, []byte(`{"providerId":"`+created.Data.ID+`"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"cost_usd":0.001`)) || !bytes.Contains(payload, []byte(`"count":1`)) {
		t.Fatalf("pollinations usage failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/usage/daily?days=7", ownerToken, []byte(`{"providerId":"`+created.Data.ID+`"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"requests":3`)) {
		t.Fatalf("pollinations daily usage failed: %d %s", status, payload)
	}
	// Models feed the provider form dropdown.
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/models", ownerToken, []byte(`{"providerId":"`+created.Data.ID+`"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"id":"openai"`)) || !bytes.Contains(payload, []byte(`"id":"flux"`)) {
		t.Fatalf("pollinations models failed: %d %s", status, payload)
	}
	// A missing/unknown provider is a 400.
	status, _ = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/account", ownerToken, []byte(`{"providerId":"`+uuid.NewString()+`"}`))
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown provider, got %d", status)
	}

	// Upstream 402 (balance/budget exhausted) surfaces as ADMN07005 and 429
	// (rate limited) as ADMN07006 — not the generic unreachable 502.
	payment := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"success":false,"error":{"message":"Insufficient pollen balance","code":"PAYMENT_REQUIRED"},"status":402}`))
	}))
	defer payment.Close()
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/providers", ownerToken, []byte(`{"name":"polli-broke","providerType":"pollinations","baseUrl":"`+payment.URL+`","model":"openai","apiKey":"sk-polli","isActive":false}`))
	if status != http.StatusOK {
		t.Fatalf("create payment provider failed: %d %s", status, payload)
	}
	var paymentProvider struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &paymentProvider); err != nil || paymentProvider.Data.ID == "" {
		t.Fatal("payment provider missing id", payload)
	}
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/account", ownerToken, []byte(`{"providerId":"`+paymentProvider.Data.ID+`"}`))
	if status != http.StatusPaymentRequired || !bytes.Contains(payload, []byte(`"errorCode":"ADMN07005"`)) {
		t.Fatalf("expected 402 ADMN07005 for exhausted balance, got %d %s", status, payload)
	}

	limited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"success":false,"error":{"message":"Slow down","code":"RATE_LIMITED"},"status":429}`))
	}))
	defer limited.Close()
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/providers", ownerToken, []byte(`{"name":"polli-limited","providerType":"pollinations","baseUrl":"`+limited.URL+`","model":"openai","apiKey":"sk-polli","isActive":false}`))
	if status != http.StatusOK {
		t.Fatalf("create rate-limited provider failed: %d %s", status, payload)
	}
	var limitedProvider struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &limitedProvider); err != nil || limitedProvider.Data.ID == "" {
		t.Fatal("rate-limited provider missing id", payload)
	}
	status, payload = call(t, server, http.MethodPost, "/api/v1/admin/ai/pollinations/usage/daily?days=7", ownerToken, []byte(`{"providerId":"`+limitedProvider.Data.ID+`"}`))
	if status != http.StatusTooManyRequests || !bytes.Contains(payload, []byte(`"errorCode":"ADMN07006"`)) {
		t.Fatalf("expected 429 ADMN07006 for rate limit, got %d %s", status, payload)
	}
}

func TestNotesEndpoints(t *testing.T) {
	server, db := adminTestMux(t)
	token := loginToken(t, server, "jane@example.com", "jane-pass")
	anon := loginToken(t, server, "owner@example.com", "owner-pass") // different user for isolation checks

	// Anonymous is rejected (401) on every notes route.
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		status, _ := call(t, server, method, "/api/v1/me/notes", "", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401 for anonymous %s /me/notes, got %d", method, status)
		}
	}

	// Create a note (with a source link), then list it.
	status, payload := call(t, server, http.MethodPost, "/api/v1/me/notes", token, []byte(`{"title":"Tokyo","content":"Ichiran is best.","sourceQuery":"best ramen tokyo"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"title":"Tokyo"`)) {
		t.Fatalf("create note failed: %d %s", status, payload)
	}
	var created struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &created); err != nil || created.Data.ID == 0 {
		t.Fatal("note missing id", payload)
	}

	status, payload = call(t, server, http.MethodGet, "/api/v1/me/notes", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte("Ichiran")) {
		t.Fatalf("list notes failed: %d %s", status, payload)
	}

	// Another user cannot update or delete jane's note.
	status, _ = call(t, server, http.MethodPatch, "/api/v1/me/notes/"+strconv.FormatUint(created.Data.ID, 10), anon, []byte(`{"title":"hijack","content":"nope"}`))
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user update, got %d", status)
	}
	status, _ = call(t, server, http.MethodDelete, "/api/v1/me/notes/"+strconv.FormatUint(created.Data.ID, 10), anon, nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user delete, got %d", status)
	}

	// Jane edits and then deletes her own note.
	status, payload = call(t, server, http.MethodPatch, "/api/v1/me/notes/"+strconv.FormatUint(created.Data.ID, 10), token, []byte(`{"title":"Tokyo v2","content":"Updated."}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"title":"Tokyo v2"`)) {
		t.Fatalf("update note failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodDelete, "/api/v1/me/notes/"+strconv.FormatUint(created.Data.ID, 10), token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"deleted":true`)) {
		t.Fatalf("delete note failed: %d %s", status, payload)
	}
	var count int64
	if err := db.Model(&entities.Note{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("expected no notes left, got %d (err=%v)", count, err)
	}
}

func TestTranslateEndpoints(t *testing.T) {
	server, _ := adminTestMux(t)
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")

	// The test mux builds the apps handler with an empty LibreTranslate base
	// URL, so the endpoints answer 503 TRAN01001 (not configured).
	status, payload := call(t, server, http.MethodGet, "/api/v1/translate/languages", janeToken, nil)
	if status != http.StatusServiceUnavailable || !bytes.Contains(payload, []byte(`"errorCode":"TRAN01001"`)) {
		t.Fatalf("expected 503 TRAN01001 for unconfigured translator, got %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodPost, "/api/v1/translate", janeToken, []byte(`{"q":"hello","source":"auto","target":"ja"}`))
	if status != http.StatusServiceUnavailable || !bytes.Contains(payload, []byte(`"errorCode":"TRAN01001"`)) {
		t.Fatalf("expected 503 TRAN01001 for unconfigured translate, got %d %s", status, payload)
	}
	// Anonymous is rejected before the service check (401).
	status, _ = call(t, server, http.MethodPost, "/api/v1/translate", "", []byte(`{"q":"hello","target":"ja"}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for anonymous translate, got %d", status)
	}
}

func TestManifestWebmanifest(t *testing.T) {
	server, db := adminTestMux(t)

	// The PWA manifest is served at the frontend origin with the live branding
	// and the standard installability fields (raw JSON, no envelope).
	status, payload := call(t, server, http.MethodGet, "/manifest.webmanifest", "", nil)
	if status != http.StatusOK {
		t.Fatalf("manifest status: %d %s", status, payload)
	}
	if !bytes.Contains(payload, []byte(`"name":"Intellisearch"`)) {
		t.Fatalf("manifest missing name: %s", payload)
	}
	if !bytes.Contains(payload, []byte(`"display":"standalone"`)) {
		t.Fatalf("manifest missing display: %s", payload)
	}
	if !bytes.Contains(payload, []byte("pwa-maskable-512x512.png")) {
		t.Fatalf("manifest missing maskable icon: %s", payload)
	}
	if bytes.Contains(payload, []byte(`"data":`)) {
		t.Fatalf("manifest must not be an API envelope: %s", payload)
	}

	// Branding changes flow through immediately (no-cache manifest).
	if err := db.Model(&entities.SiteSettings{}).Where("id = 1").Update("site_name", "Acme Search").Error; err != nil {
		t.Fatal(err)
	}
	status, payload = call(t, server, http.MethodGet, "/manifest.webmanifest", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"name":"Acme Search"`)) || bytes.Contains(payload, []byte("Intellisearch")) {
		t.Fatalf("manifest did not reflect branding: %d %s", status, payload)
	}
}
func TestGoogleSSOEndpoints(t *testing.T) {
	server, _ := adminTestMux(t)

	// /site advertises Google SSO as disabled when no credentials are configured
	// (the test mux builds its AuthService without Google env vars).
	status, payload := call(t, server, http.MethodGet, "/api/v1/site", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"googleSsoEnabled":false`)) {
		t.Fatalf("site endpoint did not report google SSO disabled: %d %s", status, payload)
	}

	// The OAuth start endpoint refuses with AUTH01003 instead of redirecting.
	request, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/google", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ = io.ReadAll(response.Body)
	if response.StatusCode != http.StatusServiceUnavailable || !bytes.Contains(payload, []byte(`"errorCode":"AUTH01003"`)) {
		t.Fatalf("google start without credentials: expected 503 AUTH01003, got %d %s", response.StatusCode, payload)
	}
}
