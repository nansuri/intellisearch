package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

// newAITestEnv wires the full pipeline against httptest fakes for SearXNG, the
// crawler, and the Ollama provider. searchDown makes SearXNG return 500s.
func newAITestEnv(t *testing.T, searchDown bool) (*AIService, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
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
		NewSearchService(config.Config{SearXNGBaseURL: searchServer.URL, SearXNGTimeoutMS: 2000}),
		NewCrawlService(crawlServer.URL, 5000, repositories.NewCrawlJobRepository(db)),
		NewLLMService(providerRepo, "key"),
		nil,
		3,
	)
	return service, db
}

func TestAnswerPersistsCitedResult(t *testing.T) {
	service, db := newAITestEnv(t, false)
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
	service, db := newAITestEnv(t, true)
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
	service, _ := newAITestEnv(t, false)
	result, err := service.Answer(context.Background(), AskInput{Query: "Summarize this page", URL: "https://example.com/article"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer == "" || len(result.Sources) != 0 {
		t.Fatalf("unexpected URL result %#v", result)
	}
}

func TestAnswerRejectsBlockedURL(t *testing.T) {
	service, _ := newAITestEnv(t, false)
	_, err := service.Answer(context.Background(), AskInput{Query: "Summarize this page", URL: "http://localhost:8090/admin"})
	if !errors.Is(err, ErrURLBlocked) {
		t.Fatalf("expected ErrURLBlocked, got %v", err)
	}
}

func TestAnswerRejectsEmptyQuery(t *testing.T) {
	service, _ := newAITestEnv(t, false)
	if _, err := service.Answer(context.Background(), AskInput{Query: "   "}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery, got %v", err)
	}
}
