package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

// newHistoryTestEnv wires SearchHistoryService against a fake OpenAI-compatible
// LLM that answers every chat request with replyContent. hits counts LLM calls
// (used to prove the suggestion cache avoids recomposition).
func newHistoryTestEnv(t *testing.T, replyContent string) (*SearchHistoryService, *gorm.DB, uuid.UUID, *atomic.Int32) {
	t.Helper()
	db := newTestDB(t)
	userID := uuid.New()
	historyRepo := repositories.NewSearchHistoryRepository(db)
	var hits atomic.Int32
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		encoded, _ := json.Marshal(replyContent)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"choices":[{"message":{"content":%s}}]}`, encoded)))
	}))
	t.Cleanup(llmServer.Close)
	providerRepo := repositories.NewProviderRepository(db)
	if err := providerRepo.Create(&entities.AIProvider{ID: uuid.New(), Name: "suggest-llm", ProviderType: "openai_compatible", BaseURL: llmServer.URL, Model: "gpt-4o-mini", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, SuggestionCacheHours: 1}).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSearchHistoryService(historyRepo, repositories.NewMessageRepository(db), NewLLMService(providerRepo, "key"), repositories.NewQueueConfigRepository(db))
	return service, db, userID, &hits
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

// seedHistoryWithAnswers seeds a history row whose MessageID points at an
// assistant message with the given content (mirroring AIService.Answer).
func seedHistoryWithAnswers(t *testing.T, db *gorm.DB, userID uuid.UUID, queries ...string) {
	t.Helper()
	historyRepo := repositories.NewSearchHistoryRepository(db)
	messageRepo := repositories.NewMessageRepository(db)
	now := time.Now().UTC()
	for i, q := range queries {
		sessionID := uuid.New()
		messageID := uuid.New()
		if err := messageRepo.Create(&entities.Message{ID: messageID, SessionID: sessionID, Role: entities.MessageRoleAssistant, Content: fmt.Sprintf("Answer to %q with plenty of detail.", q), Status: entities.MessageStatusCompleted, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := historyRepo.Create(&entities.SearchHistory{ID: uint64(now.UnixNano()) + uint64(i), UserID: userID, Query: q, SessionID: &sessionID, MessageID: &messageID, CreatedAt: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHistoryRecentQueriesDistinctAndNewestFirst(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, `["a"]`)
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

func TestHistoryRecentDetailedIncludesSummaries(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, `["a"]`)
	seedHistoryWithAnswers(t, db, userID, "alpha", "beta")
	items, err := service.RecentDetailed(userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// Newest first (beta seeded last).
	if items[0].Query != "beta" || items[1].Query != "alpha" {
		t.Fatalf("unexpected order %#v", items)
	}
	if items[0].Summary != `Answer to "beta" with plenty of detail.` {
		t.Fatalf("unexpected summary %q", items[0].Summary)
	}
	if items[0].SessionID == nil || items[0].MessageID == nil {
		t.Fatal("expected session and message IDs to be exposed")
	}
}

func TestHistoryRecentDetailedTruncatesLongSummary(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, `["a"]`)
	seedHistoryWithAnswers(t, db, userID, strings.Repeat("word ", 200))
	items, err := service.RecentDetailed(userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	summary := items[0].Summary
	if len([]rune(summary)) != 221 || !strings.HasSuffix(summary, "…") {
		t.Fatalf("summary should be truncated to 221 runes ending in ellipsis, got %d runes", len([]rune(summary)))
	}
}

func TestHistoryRecentDetailedSkipsMissingMessages(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, `["a"]`)
	// A row whose MessageID points at nothing must not fail the listing.
	seedHistory(t, db, userID, "orphan")
	items, err := service.RecentDetailed(userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Summary != "" {
		t.Fatalf("expected one item with empty summary, got %#v", items)
	}
}

func TestHistoryClearRemovesOnlyOwnEntries(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, `["a"]`)
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
	service, db, userID, _ := newHistoryTestEnv(t, `["How does Japan's rail pass work?", "Best time to visit Kyoto?", "Where to stay in Tokyo?"]`)
	seedHistory(t, db, userID, "japan travel", "tokyo itinerary")
	suggestions, err := service.Suggestions(context.Background(), userID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 || suggestions[0] != "How does Japan's rail pass work?" {
		t.Fatalf("unexpected suggestions %#v", suggestions)
	}
}

func TestSuggestionsToleratesFencedJSON(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, "```json\n[\"first\", \"second\", \"third\"]\n```")
	seedHistory(t, db, userID, "some query")
	suggestions, err := service.Suggestions(context.Background(), userID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 || suggestions[0] != "first" {
		t.Fatalf("unexpected fenced suggestions %#v", suggestions)
	}
}

func TestSuggestionsFallbackToLines(t *testing.T) {
	service, db, userID, _ := newHistoryTestEnv(t, "1. Follow-up one\n2. Follow-up two\n3. Follow-up three")
	seedHistory(t, db, userID, "some query")
	suggestions, err := service.Suggestions(context.Background(), userID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 || suggestions[0] != "Follow-up one" {
		t.Fatalf("unexpected line suggestions %#v", suggestions)
	}
}

func TestSuggestionsEmptyHistory(t *testing.T) {
	service, _, userID, _ := newHistoryTestEnv(t, `["a"]`)
	if _, err := service.Suggestions(context.Background(), userID, false); !errors.Is(err, ErrHistoryEmpty) {
		t.Fatalf("expected ErrHistoryEmpty, got %v", err)
	}
}

func TestSuggestionsCachedUntilTTL(t *testing.T) {
	service, db, userID, hits := newHistoryTestEnv(t, `["one", "two", "three"]`)
	seedHistory(t, db, userID, "query")
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("cached suggestion should not recompose, LLM hits = %d", hits.Load())
	}
}

func TestSuggestionsForceRefreshBypassesCache(t *testing.T) {
	service, db, userID, hits := newHistoryTestEnv(t, `["one", "two", "three"]`)
	seedHistory(t, db, userID, "query")
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Suggestions(context.Background(), userID, true); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("force refresh must recompose, LLM hits = %d", hits.Load())
	}
	// After the forced refresh the cache is repopulated.
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("post-refresh call should hit the cache, LLM hits = %d", hits.Load())
	}
}

func TestSuggestionsCacheDisabledWhenZero(t *testing.T) {
	service, db, userID, hits := newHistoryTestEnv(t, `["one", "two", "three"]`)
	if err := db.Model(&entities.AIQueueConfig{}).Where("id = 1").Update("suggestion_cache_hours", 0).Error; err != nil {
		t.Fatal(err)
	}
	seedHistory(t, db, userID, "query")
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("zero cache hours must compose every time, LLM hits = %d", hits.Load())
	}
}

func TestClearHistoryInvalidatesCachedSuggestions(t *testing.T) {
	service, db, userID, hits := newHistoryTestEnv(t, `["one", "two", "three"]`)
	seedHistory(t, db, userID, "query")
	if _, err := service.Suggestions(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Clear(userID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Suggestions(context.Background(), userID, false); !errors.Is(err, ErrHistoryEmpty) {
		t.Fatalf("after clearing, suggestions must recompose from empty history (got %v)", err)
	}
	_ = hits
}
