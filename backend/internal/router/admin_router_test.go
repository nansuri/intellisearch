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

func adminTestMux(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:admin-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&entities.User{}, &entities.SiteSettings{}, &entities.AIQueueConfig{}, &entities.AIProvider{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}, &entities.UsageLog{}, &entities.CrawlJob{}, &entities.SearchHistory{}); err != nil {
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
	authService := services.NewAuthService(repositories.NewUserRepository(db), cfg)
	userService := services.NewUserService(repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), t.TempDir())
	historyService := services.NewSearchHistoryService(repositories.NewSearchHistoryRepository(db), repositories.NewMessageRepository(db), services.NewLLMService(repositories.NewProviderRepository(db), cfg.JWTSecret), repositories.NewQueueConfigRepository(db))
	adminService := services.NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), cfg.JWTSecret, t.TempDir())
	statsService := services.NewStatsService(repositories.NewUsageLogRepository(db), repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil)
	aiHandler := handlers.NewAIHandler(fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), allowingLimiter{}, authService)
	defer aiHandler.Stop()
	adminHandler := handlers.NewAdminHandler(userService, adminService, statsService, services.NewOllamaService())
	siteService := services.NewSiteService(repositories.NewSiteRepository(db))
	mux := New("*", t.TempDir(), siteService, handlers.NewAuthHandler(authService), handlers.NewUserHandler(userService, historyService), nil, aiHandler, adminHandler, authService)
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
			Daily  []struct{ Count int64 `json:"count"` } `json:"daily"`
			Weekly []struct{ Count int64 `json:"count"` } `json:"weekly"`
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
	status, payload = call(t, server, http.MethodPatch, "/api/v1/admin/ai/queue-config", token, []byte(`{"maxConcurrent":6,"maxQueueSize":40,"requestTimeoutMs":120000,"perUserRateLimit":5}`))
	if status != http.StatusOK {
		t.Fatalf("update queue config failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/admin/ai/queue-config", token, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"maxConcurrent":6`)) {
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
	status, _ = call(t, server, http.MethodGet, "/api/v1/admin/stats/ai", token, nil)
	if status != http.StatusOK {
		t.Fatalf("ai stats failed: %d", status)
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