package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

// newAITestEnv wires the full pipeline against httptest fakes for SearXNG, the
// crawler, and the Ollama provider. searchDown makes SearXNG return 500s. The
// returned counter tracks crawler hits so tests can assert deep-read behavior.
func newAITestEnv(t *testing.T, searchDown bool) (*AIService, *gorm.DB, *int64) {
	t.Helper()
	db := newTestDB(t)
	var crawlHits int64
	providerRepo := repositories.NewProviderRepository(db)
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"content":"Synthesized answer citing [1] and [2]."}}`))
	}))
	t.Cleanup(ollamaServer.Close)
	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if searchDown {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Query().Get("categories") == "images" {
			_, _ = w.Write([]byte(`{"results":[{"title":"Photo A","url":"https://img.example.com/a","img_src":"https://img.example.com/a.jpg","thumbnail_src":"https://img.example.com/a-thumb.jpg","source":"example.com","resolution":"640x480"},{"title":"Photo B","url":"https://img.example.com/b","img_src":"https://img.example.com/b.jpg","thumbnail_src":"https://img.example.com/b-thumb.jpg"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"First Source","url":"https://one.example.com/a","content":"first snippet"},{"title":"Second Source","url":"https://two.example.com/b","content":"second snippet"}]}`))
	}))
	t.Cleanup(searchServer.Close)
	crawlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&crawlHits, 1)
		_, _ = w.Write([]byte(`{"title":"Deep Page","text":"full page text for grounding"}`))
	}))
	t.Cleanup(crawlServer.Close)

	if err := providerRepo.Create(&entities.AIProvider{ID: uuid.New(), Name: "test-ollama", ProviderType: "ollama", BaseURL: ollamaServer.URL, Model: "llama3.2", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	service := NewAIService(
		repositories.NewSessionRepository(db),
		repositories.NewMessageRepository(db),
		repositories.NewUsageLogRepository(db),
		providerRepo,
		repositories.NewUserRepository(db),
		repositories.NewSearchHistoryRepository(db),
		repositories.NewQueueConfigRepository(db),
		NewSearchService(config.Config{SearXNGBaseURL: searchServer.URL, SearXNGTimeoutMS: 2000}),
		NewCrawlService(crawlServer.URL, 5000, repositories.NewCrawlJobRepository(db)),
		NewLLMService(providerRepo, "key"),
		nil,
		3,
	)
	return service, db, &crawlHits
}

func TestAnswerPersistsCitedResult(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "  cheapest shipping to Japan  "})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer == "" || len(result.Sources) != 2 || result.Sources[0].Position != 1 {
		t.Fatalf("unexpected result %#v", result)
	}
	if result.SessionID == uuid.Nil || result.MessageID == uuid.Nil {
		t.Fatal("expected session and message ids")
	}
	var session entities.ChatSession
	if err := db.First(&session, "id = ?", result.SessionID).Error; err != nil {
		t.Fatal(err)
	}
	if session.Title != "cheapest shipping to Japan" {
		t.Fatalf("unexpected session title %q", session.Title)
	}
	var messages []entities.Message
	if err := db.Where("session_id = ?", result.SessionID).Order("created_at asc").Find(&messages).Error; err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != entities.MessageRoleUser || messages[1].Status != entities.MessageStatusCompleted {
		t.Fatalf("unexpected messages %#v", messages)
	}
	var sources []entities.SearchResult
	if err := db.Where("message_id = ?", result.MessageID).Find(&sources).Error; err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 persisted sources, got %d", len(sources))
	}
	// Image results are searched alongside the web results and returned/persisted.
	if len(result.Images) != 2 || result.Images[0].ThumbnailURL == "" || result.Images[0].Width != 640 || result.Images[0].Height != 480 {
		t.Fatalf("unexpected image results %#v", result.Images)
	}
	var images []entities.ImageResult
	if err := db.Where("message_id = ?", result.MessageID).Find(&images).Error; err != nil || len(images) != 2 {
		t.Fatalf("expected 2 persisted image results, got %d (err=%v)", len(images), err)
	}
	var usage entities.UsageLog
	if err := db.First(&usage).Error; err != nil {
		t.Fatal(err)
	}
	if usage.Status != entities.MessageStatusCompleted || usage.Query != "cheapest shipping to Japan" {
		t.Fatalf("unexpected usage log %#v", usage)
	}
}

func TestAnswerDegradesGracefullyWhenSearchDown(t *testing.T) {
	service, db, _ := newAITestEnv(t, true)
	result, err := service.Answer(context.Background(), AskInput{Query: "still answer me"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer == "" {
		t.Fatal("expected an answer without web sources")
	}
	if len(result.Sources) != 0 {
		t.Fatalf("expected no sources, got %d", len(result.Sources))
	}
	var usage entities.UsageLog
	if err := db.First(&usage).Error; err != nil {
		t.Fatal(err)
	}
	if usage.Status != entities.MessageStatusCompleted {
		t.Fatalf("expected completed usage log, got %q", usage.Status)
	}
}

func TestAnswerFromURLSubmission(t *testing.T) {
	service, _, _ := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "Summarize this page", URL: "https://example.com/article"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer == "" || len(result.Sources) != 0 {
		t.Fatalf("unexpected URL result %#v", result)
	}
}

func TestAnswerRejectsBlockedURL(t *testing.T) {
	service, _, _ := newAITestEnv(t, false)
	_, err := service.Answer(context.Background(), AskInput{Query: "Summarize this page", URL: "http://localhost:8090/admin"})
	if !errors.Is(err, ErrURLBlocked) {
		t.Fatalf("expected ErrURLBlocked, got %v", err)
	}
}

func TestAnswerRejectsEmptyQuery(t *testing.T) {
	service, _, _ := newAITestEnv(t, false)
	if _, err := service.Answer(context.Background(), AskInput{Query: "   "}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery, got %v", err)
	}
}

func TestAnswerRecordsSearchHistoryForLoggedInUsers(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	userID := uuid.New()
	if _, err := service.Answer(context.Background(), AskInput{Query: "history recording test", UserID: &userID}); err != nil {
		t.Fatal(err)
	}
	var entries []entities.SearchHistory
	if err := db.Where("user_id = ?", userID).Find(&entries).Error; err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Query != "history recording test" {
		t.Fatalf("expected one history entry, got %#v", entries)
	}
	// The entry links to the session and the assistant message that answered it
	// (the summary source), without duplicating answer text.
	if entries[0].SessionID == nil || entries[0].MessageID == nil {
		t.Fatalf("expected session and message IDs on the history entry, got %#v", entries[0])
	}
	var assistant entities.Message
	if err := db.First(&assistant, "id = ?", *entries[0].MessageID).Error; err != nil {
		t.Fatal(err)
	}
	if assistant.Content != "Synthesized answer citing [1] and [2]." {
		t.Fatalf("summary message content mismatch: %q", assistant.Content)
	}
}

func TestAnswerSkipsHistoryForURLSubmissions(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	userID := uuid.New()
	if _, err := service.Answer(context.Background(), AskInput{Query: "Summarize this page", URL: "https://example.com/article", UserID: &userID}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&entities.SearchHistory{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("URL submissions must not be recorded as history, got %d (err=%v)", count, err)
	}
}

func TestAnswerSearchModeBuildsExtractiveSummaryWithoutAI(t *testing.T) {
	service, db, crawlHits := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "best ramen in tokyo", Mode: ModeSearch})
	if err != nil {
		t.Fatal(err)
	}
	// The answer is an extractive summary composed from the SearXNG snippets —
	// no LLM synthesis, no deep-read crawling, no provider attribution.
	if !strings.Contains(result.Answer, "Here's what the top results say:") || !strings.Contains(result.Answer, "[1]") {
		t.Fatalf("expected an extractive summary citing sources, got %q", result.Answer)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("expected 2 raw sources, got %d", len(result.Sources))
	}
	if len(result.Images) != 2 {
		t.Fatalf("expected 2 image results, got %d", len(result.Images))
	}
	if atomic.LoadInt64(crawlHits) != 0 {
		t.Fatalf("search mode must not deep-read sources, got %d crawler hits", *crawlHits)
	}
	// Sources and images are persisted against the assistant message.
	var sources []entities.SearchResult
	if err := db.Where("message_id = ?", result.MessageID).Find(&sources).Error; err != nil || len(sources) != 2 {
		t.Fatalf("expected 2 persisted sources, got %d (err=%v)", len(sources), err)
	}
	// The usage log is completed but is not attributed to any AI provider.
	var usage entities.UsageLog
	if err := db.First(&usage, "query = ?", "best ramen in tokyo").Error; err != nil {
		t.Fatal(err)
	}
	if usage.Status != entities.MessageStatusCompleted || usage.ProviderID != nil {
		t.Fatalf("search mode usage log should be completed without a provider, got %#v", usage)
	}
}

func TestAnswerEnhancedModeDeepReadsSources(t *testing.T) {
	service, _, crawlHits := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "deep read me", Mode: ModeEnhanced})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer == "" {
		t.Fatal("enhanced mode must synthesize an answer")
	}
	if atomic.LoadInt64(crawlHits) == 0 {
		t.Fatal("enhanced mode must deep-read the top sources")
	}
}

func TestAnswerSearchModeStillRecordsHistory(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	userID := uuid.New()
	if _, err := service.Answer(context.Background(), AskInput{Query: "search only history", UserID: &userID, Mode: ModeSearch}); err != nil {
		t.Fatal(err)
	}
	var entries []entities.SearchHistory
	if err := db.Where("user_id = ?", userID).Find(&entries).Error; err != nil || len(entries) != 1 {
		t.Fatalf("search mode must record history, got %#v (err=%v)", entries, err)
	}
}

// TestAnswerCapsImageResults verifies the admin-configurable maxImageResults
// bounds how many image results are returned and persisted.
func TestAnswerCapsImageResults(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, MaxImageResults: 1}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := service.Answer(context.Background(), AskInput{Query: "capped images"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 1 || result.Images[0].Position != 1 {
		t.Fatalf("expected exactly 1 image (config cap), got %#v", result.Images)
	}
	var count int64
	if err := db.Model(&entities.ImageResult{}).Where("message_id = ?", result.MessageID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected 1 persisted image result, got %d (err=%v)", count, err)
	}
}

// TestAnswerBuildsMapDataForLocationAsks verifies that a "near me" query with
// a shared position geocodes the top source titles into map markers, returns
// center + markers, and persists them against the assistant message.
func TestAnswerBuildsMapDataForLocationAsks(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	// Give the service a geocoder: /reverse returns the place label, /search
	// returns one nearby point per source title.
	geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reverse":
			_, _ = w.Write([]byte(`{"display_name":"Jakarta, Indonesia","address":{"city":"Jakarta","country":"Indonesia"}}`))
		case "/search":
			_, _ = w.Write([]byte(`[{"display_name":"St Mary's Hospital, Jakarta","lat":"-6.2","lon":"106.8"}]`))
		default:
			t.Fatalf("unexpected geo path %s", r.URL.Path)
		}
	}))
	t.Cleanup(geoServer.Close)
	service.geo = NewGeoService(geoServer.URL, "test/1.0", 2000)

	location := GeoLocation{Latitude: -6.2, Longitude: 106.8}
	result, err := service.Answer(context.Background(), AskInput{Query: "hospital near me", Location: &location})
	if err != nil {
		t.Fatal(err)
	}
	if result.MapCenter == nil || result.MapCenter.Label != "Jakarta, Indonesia" {
		t.Fatalf("expected a map center with the place label, got %#v", result.MapCenter)
	}
	if len(result.MapMarkers) != 1 || result.MapMarkers[0].Latitude != -6.2 {
		t.Fatalf("expected one nearby marker, got %#v", result.MapMarkers)
	}
	var points []entities.MapPoint
	if err := db.Where("message_id = ?", result.MessageID).Order("position asc").Find(&points).Error; err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].Position != 0 || points[1].Position != 1 {
		t.Fatalf("expected center + marker persisted, got %#v", points)
	}
}

// TestAnswerSkipsMapWithoutLocation verifies that non-location asks never
// produce map data, and location asks without a shared position don't either.
func TestAnswerSkipsMapWithoutLocation(t *testing.T) {
	service, _, _ := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "hospital near me"})
	if err != nil {
		t.Fatal(err)
	}
	if result.MapCenter != nil || len(result.MapMarkers) != 0 {
		t.Fatalf("expected no map without a shared location, got %#v", result.MapCenter)
	}
	result, err = service.Answer(context.Background(), AskInput{Query: "what is the capital of france"})
	if err != nil {
		t.Fatal(err)
	}
	if result.MapCenter != nil || len(result.MapMarkers) != 0 {
		t.Fatalf("expected no map for a non-location query, got %#v", result.MapCenter)
	}
}

// TestBuildMapDataDropsFarMarkers verifies the radius filter keeps geocoded
// markers plausibly near the user and dedupes repeated coordinates.
func TestBuildMapDataDropsFarMarkers(t *testing.T) {
	geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"display_name":"Far Away","lat":"40.7128","lon":"-74.0060"}]`))
	}))
	t.Cleanup(geoServer.Close)
	geo := NewGeoService(geoServer.URL, "test/1.0", 2000)
	service := &AIService{geo: geo}
	center, markers := service.buildMapData(t.Context(), GeoLocation{Latitude: -6.2, Longitude: 106.8}, "Jakarta", []SourceItem{{Title: "NYC Hospital"}})
	if center.Label != "Jakarta" {
		t.Fatalf("unexpected center %#v", center)
	}
	if len(markers) != 0 {
		t.Fatalf("expected NYC marker dropped (outside 100km), got %#v", markers)
	}
}

// TestAnswerSkipsImagesOnFollowUp verifies follow-up asks (reusing a session)
// do not fetch or persist a second set of images — only the primary search of
// a thread gets the image grid.
func TestAnswerSkipsImagesOnFollowUp(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	first, err := service.Answer(context.Background(), AskInput{Query: "primary search"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Images) != 2 {
		t.Fatalf("primary search should return images, got %d", len(first.Images))
	}
	sessionID := first.SessionID
	followUp, err := service.Answer(context.Background(), AskInput{Query: "follow-up question", SessionID: &sessionID})
	if err != nil {
		t.Fatal(err)
	}
	if len(followUp.Images) != 0 {
		t.Fatalf("follow-up asks must not return images, got %d", len(followUp.Images))
	}
	var count int64
	if err := db.Model(&entities.ImageResult{}).Where("message_id = ?", followUp.MessageID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("follow-up must not persist images, got %d (err=%v)", count, err)
	}
}

// seedSuggestConversation inserts a session plus a completed user/assistant
// pair so SuggestFollowUps has a transcript to build on.
func seedSuggestConversation(t *testing.T, db *gorm.DB, userID *uuid.UUID) uuid.UUID {
	t.Helper()
	sessionID := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&entities.ChatSession{ID: sessionID, UserID: userID, Title: "test", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entities.Message{ID: uuid.New(), SessionID: sessionID, Role: entities.MessageRoleUser, Content: "What are shipping costs to Japan?", Status: entities.MessageStatusCompleted, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entities.Message{ID: uuid.New(), SessionID: sessionID, Role: entities.MessageRoleAssistant, Content: "Roughly $50 for a small parcel.", Status: entities.MessageStatusCompleted, CreatedAt: now.Add(time.Second)}).Error; err != nil {
		t.Fatal(err)
	}
	return sessionID
}

// repointActiveLLM points the active provider at a server returning the given
// chat content (any path), returning a hit counter so tests can assert that no
// model call happens when it shouldn't.
func repointActiveLLM(t *testing.T, db *gorm.DB, content string) *int64 {
	t.Helper()
	var hits int64
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		payload, err := json.Marshal(map[string]any{"message": map[string]any{"content": content}})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(llm.Close)
	if err := db.Model(&entities.AIProvider{}).Where("is_active = ?", true).Update("base_url", llm.URL).Error; err != nil {
		t.Fatal(err)
	}
	return &hits
}

func TestSuggestFollowUpsComposesFromConversation(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	sessionID := seedSuggestConversation(t, db, nil)
	hits := repointActiveLLM(t, db, `["Is air freight faster?","What do couriers charge?","How is insurance handled?"]`)
	items, err := service.SuggestFollowUps(context.Background(), nil, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0] != "Is air freight faster?" {
		t.Fatalf("unexpected suggestions %#v", items)
	}
	if *hits != 1 {
		t.Fatalf("expected exactly 1 LLM call, got %d", *hits)
	}
}

func TestSuggestFollowUpsEmptyConversationSkipsLLM(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	sessionID := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&entities.ChatSession{ID: sessionID, Title: "fresh", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	hits := repointActiveLLM(t, db, "[]")
	items, err := service.SuggestFollowUps(context.Background(), nil, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty suggestions, got %#v", items)
	}
	if *hits != 0 {
		t.Fatalf("empty conversation must not call the LLM, got %d hits", *hits)
	}
}

func TestSuggestFollowUpsErrors(t *testing.T) {
	service, db, _ := newAITestEnv(t, false)
	if _, err := service.SuggestFollowUps(context.Background(), nil, uuid.New()); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
	owner := uuid.New()
	sessionID := seedSuggestConversation(t, db, &owner)
	if _, err := service.SuggestFollowUps(context.Background(), nil, sessionID); !errors.Is(err, ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden for anonymous caller, got %v", err)
	}
	other := uuid.New()
	if _, err := service.SuggestFollowUps(context.Background(), &other, sessionID); !errors.Is(err, ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden for wrong owner, got %v", err)
	}
	if _, err := service.SuggestFollowUps(context.Background(), &owner, sessionID); err != nil {
		t.Fatalf("expected owner to succeed, got %v", err)
	}
}
