package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

// newHistoryTestEnv wires SearchHistoryService against a fake OpenAI-compatible
// LLM that answers every chat request with replyContent.
func newHistoryTestEnv(t *testing.T, replyContent string) (*SearchHistoryService, *gorm.DB, uuid.UUID) {
	t.Helper()
	db := newTestDB(t)
	userID := uuid.New()
	historyRepo := repositories.NewSearchHistoryRepository(db)
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoded, _ := json.Marshal(replyContent)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"choices":[{"message":{"content":%s}}]}`, encoded)))
	}))
	t.Cleanup(llmServer.Close)
	providerRepo := repositories.NewProviderRepository(db)
	if err := providerRepo.Create(&entities.AIProvider{ID: uuid.New(), Name: "suggest-llm", ProviderType: "openai_compatible", BaseURL: llmServer.URL, Model: "gpt-4o-mini", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	service := NewSearchHistoryService(historyRepo, NewLLMService(providerRepo, "key"))
	return service, db, userID
}

func seedHistory(t *testing.T, db *gorm.DB, userID uuid.UUID, queries ...string) {
	t.Helper()
	repo := repositories.NewSearchHistoryRepository(db)
	now := time.Now().UTC()
	for i, q := range queries {
		if err := repo.Create(&entities.SearchHistory{ID: uint64(now.UnixNano()) + uint64(i), UserID: userID, Query: q, CreatedAt: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHistoryRecentQueriesDistinctAndNewestFirst(t *testing.T) {
	service, db, userID := newHistoryTestEnv(t, `["a"]`)
	seedHistory(t, db, userID, "alpha", "beta", "alpha")
	queries, err := service.RecentQueries(userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || queries[0] != "alpha" || queries[1] != "beta" {
		t.Fatalf("expected [alpha beta] (alpha used most recently), got %#v", queries)
	}
	// Another user's history is never visible.
	other := uuid.New()
	seedHistory(t, db, other, "private")
	queries, err = service.RecentQueries(userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 {
		t.Fatalf("another user's history leaked: %#v", queries)
	}
}

func TestHistoryClearRemovesOnlyOwnEntries(t *testing.T) {
	service, db, userID := newHistoryTestEnv(t, `["a"]`)
	seedHistory(t, db, userID, "one", "two")
	other := uuid.New()
	seedHistory(t, db, other, "kept")
	if err := service.Clear(userID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&entities.SearchHistory{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected only the other user's entry to remain, got %d (err=%v)", count, err)
	}
}

func TestSuggestionsComposeFromHistory(t *testing.T) {
	service, db, userID := newHistoryTestEnv(t, `["How does Japan's rail pass work?", "Best time to visit Kyoto?", "Where to stay in Tokyo?"]`)
	seedHistory(t, db, userID, "japan travel", "tokyo itinerary")
	suggestions, err := service.Suggestions(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 || suggestions[0] != "How does Japan's rail pass work?" {
		t.Fatalf("unexpected suggestions %#v", suggestions)
	}
}

func TestSuggestionsToleratesFencedJSON(t *testing.T) {
	service, db, userID := newHistoryTestEnv(t, "```json\n[\"first\", \"second\", \"third\"]\n```")
	seedHistory(t, db, userID, "some query")
	suggestions, err := service.Suggestions(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 || suggestions[0] != "first" {
		t.Fatalf("unexpected fenced suggestions %#v", suggestions)
	}
}

func TestSuggestionsFallbackToLines(t *testing.T) {
	service, db, userID := newHistoryTestEnv(t, "1. Follow-up one\n2. Follow-up two\n3. Follow-up three")
	seedHistory(t, db, userID, "some query")
	suggestions, err := service.Suggestions(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 || suggestions[0] != "Follow-up one" {
		t.Fatalf("unexpected line suggestions %#v", suggestions)
	}
}

func TestSuggestionsEmptyHistory(t *testing.T) {
	service, _, userID := newHistoryTestEnv(t, `["a"]`)
	if _, err := service.Suggestions(context.Background(), userID); !errors.Is(err, ErrHistoryEmpty) {
		t.Fatalf("expected ErrHistoryEmpty, got %v", err)
	}
}
