package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var ErrHistoryEmpty = errors.New("no search history yet")

// HistoryItem is the API shape for one search-history row, including an
// on-demand summary (the assistant message that answered the search,
// truncated — the full answer stays in the session thread).
type HistoryItem struct {
	ID        uint64     `json:"id"`
	UserID    uuid.UUID  `json:"userId"`
	Query     string     `json:"query"`
	SessionID *uuid.UUID `json:"sessionId,omitempty"`
	MessageID *uuid.UUID `json:"messageId,omitempty"`
	Summary   string     `json:"summary,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// SearchHistoryService owns the per-user search-history feature: the "recent
// searches" chips on the main page, the history page with summaries, the
// AI-composed suggestions, and clearing. Suggestions are expensive (a model
// call), so they are cached per user for a configurable number of hours
// (ai_queue_config.suggestion_cache_hours, editable from the Owner Control
// Panel); 0 disables the cache and always composes fresh.
type SearchHistoryService struct {
	history     *repositories.SearchHistoryRepository
	messages    *repositories.MessageRepository
	llm         *LLMService
	queueConfig *repositories.QueueConfigRepository

	mu    sync.Mutex
	cache map[uuid.UUID]cachedSuggestions
}

type cachedSuggestions struct {
	at    time.Time
	items []string
}

func NewSearchHistoryService(history *repositories.SearchHistoryRepository, messages *repositories.MessageRepository, llm *LLMService, queueConfig *repositories.QueueConfigRepository) *SearchHistoryService {
	return &SearchHistoryService{history: history, messages: messages, llm: llm, queueConfig: queueConfig, cache: make(map[uuid.UUID]cachedSuggestions)}
}

// Recent returns the user's most recent history entries, newest first.
func (s *SearchHistoryService) Recent(userID uuid.UUID, limit int) ([]entities.SearchHistory, error) {
	return s.history.Recent(userID, limit)
}

// RecentQueries returns the user's most recently used queries without duplicates.
func (s *SearchHistoryService) RecentQueries(userID uuid.UUID, limit int) ([]string, error) {
	return s.history.RecentDistinct(userID, limit)
}

// RecentDetailed returns the user's recent history with on-demand summaries.
// The summary is the assistant message that answered the search, truncated to
// a compact preview — nothing extra is stored (only the message ID is kept in
// search_history). Search-only asks have no AI answer, so their summary is empty.
func (s *SearchHistoryService) RecentDetailed(userID uuid.UUID, limit int) ([]HistoryItem, error) {
	entries, err := s.history.Recent(userID, limit)
	if err != nil {
		return nil, err
	}
	messageIDs := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		if entry.MessageID != nil {
			messageIDs = append(messageIDs, *entry.MessageID)
		}
	}
	summaries := map[uuid.UUID]string{}
	if len(messageIDs) > 0 {
		if summaries, err = s.messages.Summaries(messageIDs); err != nil {
			return nil, err
		}
	}
	items := make([]HistoryItem, 0, len(entries))
	for _, entry := range entries {
		item := HistoryItem{ID: entry.ID, UserID: entry.UserID, Query: entry.Query, SessionID: entry.SessionID, MessageID: entry.MessageID, CreatedAt: entry.CreatedAt}
		if entry.MessageID != nil {
			item.Summary = truncateSummary(summaries[*entry.MessageID])
		}
		items = append(items, item)
	}
	return items, nil
}

// Clear removes all of the user's history and drops their cached suggestions
// (suggestions derived from the now-deleted history must not linger).
func (s *SearchHistoryService) Clear(userID uuid.UUID) error {
	if err := s.history.Clear(userID); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.cache, userID)
	s.mu.Unlock()
	return nil
}

// Suggestions returns AI-composed follow-up questions based on the user's
// recent search history, cached per user for suggestion_cache_hours. force
// bypasses the cache (the main page's ↻ button). A missing history returns
// ErrHistoryEmpty; provider failures return the error so the handler can
// degrade to an empty list.
func (s *SearchHistoryService) Suggestions(ctx context.Context, userID uuid.UUID, force bool) ([]string, error) {
	ttl := s.suggestionTTL()
	if !force && ttl > 0 {
		s.mu.Lock()
		if cached, ok := s.cache[userID]; ok && time.Since(cached.at) < ttl {
			s.mu.Unlock()
			return cached.items, nil
		}
		s.mu.Unlock()
	}

	queries, err := s.history.RecentDistinct(userID, 10)
	if err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return nil, ErrHistoryEmpty
	}
	system := `You are a research assistant. Suggest follow-up questions for a user based on their recent search history. Return ONLY a JSON array of strings with 3 specific, self-contained questions, for example ["question one", "question two", "question three"]. No markdown, no numbering, no commentary.`
	userPrompt := "Recent searches: " + strings.Join(queries, " | ") + ". Suggest 3 related questions the user may want to explore next."
	generated, err := s.llm.Generate(ctx, system, []ChatMessage{{Role: "user", Content: userPrompt}}, GenerateOptions{Temperature: 0.4, MaxTokens: 200})
	if err != nil {
		return nil, err
	}
	items := parseSuggestions(generated.Content)
	if ttl > 0 {
		s.mu.Lock()
		s.cache[userID] = cachedSuggestions{at: time.Now(), items: items}
		s.mu.Unlock()
	}
	return items, nil
}

// suggestionTTL reads the cache window from ai_queue_config. 0 or an unreadable
// config disables the cache (always compose fresh).
func (s *SearchHistoryService) suggestionTTL() time.Duration {
	config, err := s.queueConfig.Get()
	if err != nil || config.SuggestionCacheHours <= 0 {
		return 0
	}
	return time.Duration(config.SuggestionCacheHours) * time.Hour
}

// truncateSummary shortens an AI answer to a compact preview for the history
// list. The full answer is never duplicated here — it stays in the session.
func truncateSummary(content string) string {
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if len(runes) <= 220 {
		return content
	}
	return string(runes[:220]) + "…"
}

// parseSuggestions extracts a list of suggestion strings from the LLM reply,
// tolerating JSON arrays, fenced code blocks, and plain line lists.
func parseSuggestions(raw string) []string {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimPrefix(text, "json")
	text = strings.TrimSpace(text)
	if idx := strings.LastIndex(text, "```"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}
	var array []string
	if err := json.Unmarshal([]byte(text), &array); err == nil {
		out := make([]string, 0, len(array))
		for _, q := range array {
			q = strings.TrimSpace(q)
			if q != "" {
				out = append(out, q)
			}
		}
		return out
	}
	var out []string
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(strings.Trim(scanner.Text(), "-[]0123456789. "))
		if line != "" {
			out = append(out, line)
		}
		if len(out) >= 5 {
			break
		}
	}
	return out
}
