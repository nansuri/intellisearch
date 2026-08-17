package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var ErrHistoryEmpty = errors.New("no search history yet")

// SearchHistoryService owns the per-user search-history feature: the "recent
// searches" chips on the main page, the history list in Account Settings, the
// AI-composed suggestions, and clearing.
type SearchHistoryService struct {
	history *repositories.SearchHistoryRepository
	llm     *LLMService
}

func NewSearchHistoryService(history *repositories.SearchHistoryRepository, llm *LLMService) *SearchHistoryService {
	return &SearchHistoryService{history: history, llm: llm}
}

// Recent returns the user's most recent history entries, newest first.
func (s *SearchHistoryService) Recent(userID uuid.UUID, limit int) ([]entities.SearchHistory, error) {
	return s.history.Recent(userID, limit)
}

// RecentQueries returns the user's most recently used queries without duplicates.
func (s *SearchHistoryService) RecentQueries(userID uuid.UUID, limit int) ([]string, error) {
	return s.history.RecentDistinct(userID, limit)
}

// Clear removes all of the user's history.
func (s *SearchHistoryService) Clear(userID uuid.UUID) error {
	return s.history.Clear(userID)
}

// Suggestions asks the active LLM to compose follow-up questions based on the
// user's recent search history. A missing history or provider failure returns
// an empty suggestion list (the UI simply hides the row); ErrHistoryEmpty is
// returned so the handler can skip logging a provider error for empty history.
func (s *SearchHistoryService) Suggestions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	queries, err := s.history.RecentDistinct(userID, 10)
	if err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return nil, ErrHistoryEmpty
	}
	system := `You are a research assistant. Suggest follow-up questions for a user based on their recent search history. Return ONLY a JSON array of strings with 3 specific, self-contained questions, for example ["question one", "question two", "question three"]. No markdown, no numbering, no commentary.`
	userPrompt := "Recent searches: " + strings.Join(queries, " | ") + ". Suggest 3 related questions the user may want to explore next."
	raw, err := s.llm.Generate(ctx, system, []ChatMessage{{Role: "user", Content: userPrompt}}, GenerateOptions{Temperature: 0.4, MaxTokens: 200})
	if err != nil {
		return nil, err
	}
	return parseSuggestions(raw), nil
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
