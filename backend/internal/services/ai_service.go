package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var (
	ErrInvalidQuery     = errors.New("invalid question")
	ErrQueueFull        = errors.New("ai queue full")
	ErrQuotaExceeded    = errors.New("daily question quota exceeded")
	ErrRateLimited      = errors.New("rate limit exceeded")
	ErrAnonymousLimit   = errors.New("anonymous guest AI allowance used")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionForbidden = errors.New("session access denied")
)

const maxQueryLength = 2000
const maxPageTextLength = 8000

// Ask modes: enhanced runs the full pipeline (SearXNG + crawler + LLM
// synthesis); search returns raw web results without any AI work.
const (
	ModeEnhanced = "enhanced"
	ModeSearch   = "search"
)

// AskInput describes one AI job submitted by the handler.
type AskInput struct {
	Query     string
	SessionID *uuid.UUID
	UserID    *uuid.UUID // nil for anonymous users
	URL       string     // non-empty for URL-submission asks
	IP        string     // fallback identity for anonymous rate limiting
	Location  *GeoLocation
	Mode      string     // ModeEnhanced (default) or ModeSearch
}

// AskResult is the envelope payload returned to the client. VisitorID is set
// only for anonymous callers on their first (allowed) ask, so the frontend can
// persist the issued guest token.
type AskResult struct {
	SessionID uuid.UUID   `json:"sessionId"`
	MessageID uuid.UUID   `json:"messageId"`
	Answer    string      `json:"answer"`
	Sources   []SourceItem `json:"sources"`
	VisitorID *uuid.UUID  `json:"visitorId,omitempty"`
}

// AIService runs the full ask pipeline: persist session/messages/usage logs,
// search the web, deep-read top sources, and synthesize a cited answer.
type AIService struct {
	sessions  *repositories.SessionRepository
	messages  *repositories.MessageRepository
	usageLogs *repositories.UsageLogRepository
	providers *repositories.ProviderRepository
	users     *repositories.UserRepository
	history   *repositories.SearchHistoryRepository // nil disables recording (tests)
	search    *SearchService
	crawl     *CrawlService
	llm       *LLMService
	geo       *GeoService
	crawlTopN int
}

func NewAIService(sessions *repositories.SessionRepository, messages *repositories.MessageRepository, usageLogs *repositories.UsageLogRepository, providers *repositories.ProviderRepository, users *repositories.UserRepository, history *repositories.SearchHistoryRepository, search *SearchService, crawl *CrawlService, llm *LLMService, geo *GeoService, crawlTopN int) *AIService {
	return &AIService{sessions: sessions, messages: messages, usageLogs: usageLogs, providers: providers, users: users, history: history, search: search, crawl: crawl, llm: llm, geo: geo, crawlTopN: crawlTopN}
}

// Answer executes one ask job. It always persists a user message, an assistant
// message, and a usage log; fatal errors mark them failed and are returned to
// the handler for envelope mapping.
func (s *AIService) Answer(ctx context.Context, input AskInput) (AskResult, error) {
	query := SanitizeQuery(input.Query)
	if query == "" {
		return AskResult{}, ErrInvalidQuery
	}
	started := time.Now().UTC()
	result := AskResult{}
	session, err := s.resolveSession(input, query)
	if err != nil {
		return result, err
	}
	result.SessionID = session.ID

	userMessage := entities.Message{ID: uuid.New(), SessionID: session.ID, Role: entities.MessageRoleUser, Content: query, Status: entities.MessageStatusCompleted, CreatedAt: started}
	if err := s.messages.Create(&userMessage); err != nil {
		return result, err
	}
	assistantMessage := entities.Message{ID: uuid.New(), SessionID: session.ID, Role: entities.MessageRoleAssistant, Content: "", Status: entities.MessageStatusQueued, CreatedAt: time.Now().UTC()}
	if err := s.messages.Create(&assistantMessage); err != nil {
		return result, err
	}
	result.MessageID = assistantMessage.ID

	usageLog := entities.UsageLog{ID: uint64(time.Now().UnixNano()), UserID: input.UserID, Query: query, Status: entities.MessageStatusQueued, CreatedAt: started}
	if err := s.usageLogs.Create(&usageLog); err != nil {
		return result, err
	}

	// Record the search in the user's history so the main page can show recent
	// searches and AI-composed suggestions, and the history page can show
	// summaries. Only the session/message IDs are stored (never the answer
	// text — summaries are fetched from the messages table on demand). URL
	// submissions use a synthetic query, so they are skipped. Best-effort: a
	// history write failure never fails the search itself.
	if s.history != nil && input.UserID != nil && input.URL == "" {
		_ = s.history.Create(&entities.SearchHistory{ID: uint64(time.Now().UnixNano()), UserID: *input.UserID, Query: query, SessionID: &session.ID, MessageID: &assistantMessage.ID, CreatedAt: time.Now().UTC()})
	}

	provider, providerErr := s.providers.Active()
	fail := func(cause error) (AskResult, error) {
		assistantMessage.Status = entities.MessageStatusFailed
		_ = s.messages.Update(&assistantMessage)
		code, message := CodeForError(cause), SanitizedErrorMessage(cause)
		usageLog.LatencyMS = int(time.Since(started).Milliseconds())
		usageLog.Status = entities.MessageStatusFailed
		usageLog.ErrorCode = &code
		usageLog.ErrorMessage = &message
		if providerErr == nil {
			usageLog.ProviderID = &provider.ID
		}
		_ = s.usageLogs.Update(&usageLog)
		return AskResult{}, cause
	}

	// Mark the assistant message as streaming while the pipeline runs.
	assistantMessage.Status = entities.MessageStatusStreaming
	_ = s.messages.Update(&assistantMessage)

	sources := []SourceItem{}
	var promptSources []string
	locationNote := ""
	if input.URL != "" {
		promptSources, err = s.collectFromURL(ctx, input, &usageLog)
	} else {
		var searchQuery string
		searchQuery, locationNote = BuildLocationContext(ctx, s.geo, query, input.Location)
		// Search-only mode skips the deep-read (no crawler work) — results are raw.
		sources, promptSources, err = s.collectFromSearch(ctx, searchQuery, assistantMessage.ID, input.Mode != ModeSearch)
	}
	if err != nil {
		return fail(err)
	}

	// "Ask" (search mode) stops here: no LLM synthesis, just the web results.
	// The assistant message still exists (empty) so session history and source
	// cards stay consistent, and the usage log records a completed non-AI ask.
	if input.Mode == ModeSearch {
		assistantMessage.Status = entities.MessageStatusCompleted
		usageLog.LatencyMS = int(time.Since(started).Milliseconds())
		usageLog.Status = entities.MessageStatusCompleted
		_ = s.messages.Update(&assistantMessage)
		_ = s.usageLogs.Update(&usageLog)
		result.Sources = sources
		return result, nil
	}

	system := "You are a research assistant. Answer the question concisely and accurately. Format your answer in Markdown so it renders well: use short headings, bullet lists, and bold highlights where they help readability, and use emoji sparingly to make key points stand out. When you use information from the provided sources, cite them inline as [1], [2], etc., matching the numbered source list. If a source includes a relevant image, you may embed it with markdown image syntax. If no sources are provided, answer from general knowledge and note that you could not find web sources."
	userPrompt := query
	if locationNote != "" {
		userPrompt += "\n\n[Location context: " + locationNote + "]"
	}
	if len(promptSources) > 0 {
		userPrompt += "\n\nAvailable sources:\n" + strings.Join(promptSources, "\n\n")
	}
	chat := s.conversationContext(session, userMessage.ID)
	answer, err := s.llm.GenerateWith(ctx, provider, system, append(chat, ChatMessage{Role: "user", Content: userPrompt}), GenerateOptions{})
	if err != nil {
		return fail(err)
	}

	assistantMessage.Content = answer
	assistantMessage.Status = entities.MessageStatusCompleted
	usageLog.LatencyMS = int(time.Since(started).Milliseconds())
	usageLog.Status = entities.MessageStatusCompleted
	if providerErr == nil {
		usageLog.ProviderID = &provider.ID
	}
	if err := s.messages.Update(&assistantMessage); err != nil {
		return result, err
	}
	if err := s.usageLogs.Update(&usageLog); err != nil {
		return result, err
	}
	result.Answer = answer
	result.Sources = sources
	return result, nil
}

// resolveSession reuses an existing session for follow-ups or creates a new one.
func (s *AIService) resolveSession(input AskInput, query string) (entities.ChatSession, error) {
	if input.SessionID != nil {
		session, err := s.sessions.Get(*input.SessionID)
		if err != nil {
			return entities.ChatSession{}, ErrSessionNotFound
		}
		if session.UserID != nil && (input.UserID == nil || *session.UserID != *input.UserID) {
			return entities.ChatSession{}, ErrSessionForbidden
		}
		session.UpdatedAt = time.Now().UTC()
		_ = s.sessions.Update(&session)
		return session, nil
	}
	session := entities.ChatSession{ID: uuid.New(), UserID: input.UserID, Title: query, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.sessions.Create(&session); err != nil {
		return entities.ChatSession{}, err
	}
	return session, nil
}

// conversationContext carries the recent Q&A history into the LLM chat.
func (s *AIService) conversationContext(session entities.ChatSession, currentUserMessageID uuid.UUID) []ChatMessage {
	messages, err := s.sessions.Messages(session.ID)
	if err != nil {
		return nil
	}
	chat := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		if message.ID == currentUserMessageID || message.Status != entities.MessageStatusCompleted {
			continue
		}
		role := message.Role
		if role == entities.MessageRoleUser {
			role = "user"
		} else if role == entities.MessageRoleAssistant {
			role = "assistant"
		} else {
			continue
		}
		chat = append(chat, ChatMessage{Role: role, Content: message.Content})
	}
	if len(chat) > 6 {
		chat = chat[len(chat)-6:]
	}
	return chat
}

// collectFromSearch queries SearXNG, persists the source cards, and optionally
// deep-reads the top pages (only for enhanced mode). Search failures degrade
// gracefully: the LLM still answers, without web sources.
func (s *AIService) collectFromSearch(ctx context.Context, query string, messageID uuid.UUID, deepRead bool) ([]SourceItem, []string, error) {
	items, err := s.search.Search(ctx, query)
	if err != nil {
		return []SourceItem{}, nil, nil
	}
	if deepRead {
		if err := s.deepRead(ctx, items); err != nil {
			return []SourceItem{}, nil, err
		}
	}
	sources := make([]entities.SearchResult, 0, len(items))
	promptSources := make([]string, 0, len(items))
	for _, item := range items {
		sources = append(sources, entities.SearchResult{ID: uuid.New(), MessageID: messageID, Position: item.Position, Title: item.Title, URL: item.URL, Domain: item.Domain, Snippet: item.Snippet})
		promptSources = append(promptSources, fmt.Sprintf("[%d] %s — %s\n%s", item.Position, item.Title, item.URL, item.Snippet))
	}
	if err := s.messages.CreateSources(sources); err != nil {
		return nil, nil, err
	}
	return items, promptSources, nil
}

// deepRead fetches the top N source pages in parallel, best-effort.
func (s *AIService) deepRead(ctx context.Context, items []SourceItem) error {
	limit := s.crawlTopN
	if limit > len(items) {
		limit = len(items)
	}
	if limit == 0 {
		return nil
	}
	var wg sync.WaitGroup
	texts := make([]string, len(items))
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			page, err := s.crawl.FetchPage(ctx, items[index].URL, nil)
			if err != nil {
				return
			}
			texts[index] = page.Text
		}(i)
	}
	wg.Wait()
	for i, text := range texts {
		if text != "" {
			items[i].Snippet = text[:min(len(text), maxPageTextLength)]
		}
	}
	return nil
}

// collectFromURL validates and crawls a user-submitted URL, then builds the
// prompt from the page content alone.
func (s *AIService) collectFromURL(ctx context.Context, input AskInput, usageLog *entities.UsageLog) ([]string, error) {
	page, err := s.crawl.FetchPage(ctx, input.URL, input.UserID)
	if err != nil {
		return nil, err
	}
	text := page.Text
	if len(text) > maxPageTextLength {
		text = text[:maxPageTextLength]
	}
	title := page.Title
	if title == "" {
		title = input.URL
	}
	return []string{fmt.Sprintf("Page title: %s\n\n%s", title, text)}, nil
}

// SanitizeQuery trims and truncates user input before it is persisted or sent
// to the LLM (log-hygiene: no control characters or unbounded length).
func SanitizeQuery(query string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, query)
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) > maxQueryLength {
		cleaned = cleaned[:maxQueryLength]
	}
	return cleaned
}

// CodeForError maps an AI pipeline error to its typed error-code constant.
func CodeForError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidQuery):
		return "AISY01004"
	case errors.Is(err, ErrQueueFull):
		return "AISY02001"
	case errors.Is(err, ErrRateLimited):
		return "AISY02002"
	case errors.Is(err, ErrQuotaExceeded):
		return "AISY02003"
	case errors.Is(err, ErrAnonymousLimit):
		return "AISY02004"
	case errors.Is(err, ErrURLInvalid):
		return "AISY03003"
	case errors.Is(err, ErrURLBlocked):
		return "AISY03002"
	case errors.Is(err, ErrCrawlFailed):
		return "AISY03004"
	case errors.Is(err, ErrAITimeout):
		return "AISY01002"
	case errors.Is(err, ErrAIProviderError):
		return "AISY01003"
	case errors.Is(err, ErrAIUnavailable):
		return "AISY01001"
	default:
		return "AISY01001"
	}
}

// SanitizedErrorMessage returns the sanitized, user-facing copy for an error.
// Internal causes are logged by the handler; this string is safe for clients.
func SanitizedErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidQuery):
		return "Ask a question first."
	case errors.Is(err, ErrQueueFull):
		return "We're busy right now — try again in a moment."
	case errors.Is(err, ErrRateLimited):
		return "You're asking too quickly — slow down and try again in a moment."
	case errors.Is(err, ErrQuotaExceeded):
		return "You've reached today's question limit — try again tomorrow."
	case errors.Is(err, ErrAnonymousLimit):
		return "Guests get one AI search — sign in to continue."
	case errors.Is(err, ErrURLInvalid):
		return "That URL is not valid."
	case errors.Is(err, ErrURLBlocked):
		return "That URL is not allowed — internal or private addresses are blocked."
	case errors.Is(err, ErrCrawlFailed):
		return "We couldn't read that page — it may be unavailable or unreadable."
	case errors.Is(err, ErrAITimeout):
		return "The answer took too long — please try again."
	case errors.Is(err, ErrAIProviderError):
		return "The AI service returned an error — please try again."
	default:
		return "The AI service is temporarily unavailable — please try again."
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
