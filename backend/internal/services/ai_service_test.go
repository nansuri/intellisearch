package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

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

func TestAnswerSearchModeReturnsRawResultsWithoutAI(t *testing.T) {
	service, db, crawlHits := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "best ramen in tokyo", Mode: ModeSearch})
	if err != nil {
		t.Fatal(err)
	}
	// Raw results only: no synthesized answer, no deep-read crawling.
	if result.Answer != "" {
		t.Fatalf("search mode must not synthesize an answer, got %q", result.Answer)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("expected 2 raw sources, got %d", len(result.Sources))
	}
	if atomic.LoadInt64(crawlHits) != 0 {
		t.Fatalf("search mode must not deep-read sources, got %d crawler hits", *crawlHits)
	}
	// Sources are still persisted against the (empty) assistant message.
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
