package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

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
	Mode      string // ModeEnhanced (default) or ModeSearch
}

// MapPoint is a map marker: position 0 is the map center (the user's
// location, reverse-geocoded to a label) and positions 1..N are nearby
// results geocoded from the top source titles.
type MapPoint struct {
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AskResult is the envelope payload returned to the client. VisitorID is set
// only for anonymous callers on their first (allowed) ask, so the frontend can
// persist the issued guest token. MapCenter/MapMarkers are only populated for
// location-aware asks ("near me") where the user shared their position.
type AskResult struct {
	SessionID  uuid.UUID    `json:"sessionId"`
	MessageID  uuid.UUID    `json:"messageId"`
	Answer     string       `json:"answer"`
	Sources    []SourceItem `json:"sources"`
	Images     []ImageItem  `json:"images"`
	MapCenter  *MapPoint    `json:"mapCenter,omitempty"`
	MapMarkers []MapPoint   `json:"mapMarkers,omitempty"`
	VisitorID  *uuid.UUID   `json:"visitorId,omitempty"`
}

// AIService runs the full ask pipeline: persist session/messages/usage logs,
// search the web, deep-read top sources, and synthesize a cited answer.
type AIService struct {
	sessions    *repositories.SessionRepository
	messages    *repositories.MessageRepository
	usageLogs   *repositories.UsageLogRepository
	providers   *repositories.ProviderRepository
	users       *repositories.UserRepository
	history     *repositories.SearchHistoryRepository // nil disables recording (tests)
	queueConfig *repositories.QueueConfigRepository   // nil falls back to the default image cap
	search      *SearchService
	crawl       *CrawlService
	llm         *LLMService
	geo         *GeoService
	crawlTopN   int
}

func NewAIService(sessions *repositories.SessionRepository, messages *repositories.MessageRepository, usageLogs *repositories.UsageLogRepository, providers *repositories.ProviderRepository, users *repositories.UserRepository, history *repositories.SearchHistoryRepository, queueConfig *repositories.QueueConfigRepository, search *SearchService, crawl *CrawlService, llm *LLMService, geo *GeoService, crawlTopN int) *AIService {
	return &AIService{sessions: sessions, messages: messages, usageLogs: usageLogs, providers: providers, users: users, history: history, queueConfig: queueConfig, search: search, crawl: crawl, llm: llm, geo: geo, crawlTopN: crawlTopN}
}

// maxImageResults returns the admin-configurable cap on image results per
// search, falling back to 20 when the queue-config row is unavailable.
func (s *AIService) maxImageResults() int {
	if s.queueConfig != nil {
		if config, err := s.queueConfig.Get(); err == nil {
			return config.MaxImageResults
		}
	}
	return 20
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
	images := []ImageItem{}
	var promptSources []string
	locationNote := ""
	placeLabel := ""
	if input.URL != "" {
		promptSources, err = s.collectFromURL(ctx, input, &usageLog)
	} else {
		var searchQuery string
		searchQuery, locationNote, placeLabel = BuildLocationContext(ctx, s.geo, query, input.Location)
		// Search-only mode skips the deep-read (no crawler work). Follow-up asks
		// (reusing a session) skip the image search too, so only the primary
		// search of a thread fetches images.
		sources, promptSources, images, err = s.collectFromSearch(ctx, searchQuery, assistantMessage.ID, input.Mode != ModeSearch, input.SessionID == nil)
	}
	if err != nil {
		return fail(err)
	}

	// Location-aware asks ("hospital near me") with a shared position also get
	// a map: the user's location as the center plus nearby results geocoded
	// from the top source titles. Best-effort — a geocoding failure only means
	// fewer markers, never a failed ask.
	if input.Location != nil && NeedsLocationContext(query) && placeLabel != "" {
		center, markers := s.buildMapData(ctx, *input.Location, placeLabel, sources)
		result.MapCenter = &center
		result.MapMarkers = markers
		rows := make([]entities.MapPoint, 0, len(markers)+1)
		rows = append(rows, entities.MapPoint{MessageID: assistantMessage.ID, Position: 0, Label: center.Label, Latitude: center.Latitude, Longitude: center.Longitude})
		for index, marker := range markers {
			rows = append(rows, entities.MapPoint{MessageID: assistantMessage.ID, Position: index + 1, Label: marker.Label, Latitude: marker.Latitude, Longitude: marker.Longitude})
		}
		if err := s.messages.CreateMapPoints(rows); err != nil {
			logrus.WithError(err).Warn("map points persist failed; continuing without them")
		}
	}

	// "Ask" (search mode) stops here: no LLM synthesis and no deep-read — the
	// answer is an extractive summary composed from the SearXNG snippets, so it
	// costs zero AI tokens. The usage log records a completed non-AI ask.
	if input.Mode == ModeSearch {
		assistantMessage.Content = buildSearchSummary(sources)
		assistantMessage.Status = entities.MessageStatusCompleted
		usageLog.LatencyMS = int(time.Since(started).Milliseconds())
		usageLog.Status = entities.MessageStatusCompleted
		_ = s.messages.Update(&assistantMessage)
		_ = s.usageLogs.Update(&usageLog)
		result.Answer = assistantMessage.Content
		result.Sources = sources
		result.Images = images
		return result, nil
	}

	system := "You are a research assistant. Answer the question concisely and accurately. Format your answer in Markdown so it renders well: use short headings, bullet lists, and bold highlights where they help readability, and use emoji sparingly to make key points stand out. When you use information from the provided sources, cite them inline as [1], [2], etc., matching the numbered source list. If a source includes a relevant image, you may embed it with markdown image syntax. If no sources are provided, answer from general knowledge and note that you could not find web sources.\n\nYou can also generate visuals inside your answer — no image files needed, they render directly:\n- Diagrams, flowcharts, UML, sequence diagrams, timelines, and charts: use a fenced code block tagged mermaid (for example: mermaid flowchart TD ... with the fences).\n- Simple charts and art: use ASCII art inside a plain fenced code block, or a Markdown table.\nUse these whenever a chart or diagram makes the answer clearer (e.g. comparing options, showing a flow, architecture, or process)."
	userPrompt := query
	if locationNote != "" {
		userPrompt += "\n\n[Location context: " + locationNote + "]"
	}
	if len(promptSources) > 0 {
		userPrompt += "\n\nAvailable sources:\n" + strings.Join(promptSources, "\n\n")
	}
	chat := s.conversationContext(session, userMessage.ID)
	generated, err := s.llm.GenerateWith(ctx, provider, system, append(chat, ChatMessage{Role: "user", Content: userPrompt}), GenerateOptions{})
	if err != nil {
		return fail(err)
	}

	assistantMessage.Content = generated.Content
	assistantMessage.Status = entities.MessageStatusCompleted
	usageLog.LatencyMS = int(time.Since(started).Milliseconds())
	usageLog.Status = entities.MessageStatusCompleted
	usageLog.InputTokens = generated.InputTokens
	usageLog.OutputTokens = generated.OutputTokens
	usageLog.GenerateMS = int(generated.Duration.Milliseconds())
	if providerErr == nil {
		usageLog.ProviderID = &provider.ID
	}
	if err := s.messages.Update(&assistantMessage); err != nil {
		return result, err
	}
	if err := s.usageLogs.Update(&usageLog); err != nil {
		return result, err
	}
	result.Answer = generated.Content
	result.Sources = sources
	result.Images = images
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

// collectFromSearch queries SearXNG (web + images), persists the source cards
// and image results, and optionally deep-reads the top pages (only for
// enhanced mode). Search failures degrade gracefully: the LLM still answers,
// without web sources; an image-search failure never fails the ask. withImages
// gates the image search (follow-up asks pass false) and the image count is
// capped by the admin-configurable maxImageResults.
func (s *AIService) collectFromSearch(ctx context.Context, query string, messageID uuid.UUID, deepRead, withImages bool) ([]SourceItem, []string, []ImageItem, error) {
	items, err := s.search.Search(ctx, query)
	if err != nil {
		return []SourceItem{}, nil, []ImageItem{}, nil
	}
	if deepRead {
		if err := s.deepRead(ctx, items); err != nil {
			return []SourceItem{}, nil, []ImageItem{}, err
		}
	}
	sources := make([]entities.SearchResult, 0, len(items))
	promptSources := make([]string, 0, len(items))
	for _, item := range items {
		sources = append(sources, entities.SearchResult{ID: uuid.New(), MessageID: messageID, Position: item.Position, Title: item.Title, URL: item.URL, Domain: item.Domain, Snippet: item.Snippet})
		promptSources = append(promptSources, fmt.Sprintf("[%d] %s — %s\n%s", item.Position, item.Title, item.URL, item.Snippet))
	}
	if err := s.messages.CreateSources(sources); err != nil {
		return nil, nil, []ImageItem{}, err
	}
	if !withImages {
		return items, promptSources, []ImageItem{}, nil
	}
	images, imageErr := s.search.SearchImages(ctx, query, s.maxImageResults())
	if imageErr != nil {
		logrus.WithError(imageErr).Warn("image search failed; continuing without images")
	} else if len(images) > 0 {
		rows := make([]entities.ImageResult, 0, len(images))
		for _, image := range images {
			rows = append(rows, entities.ImageResult{MessageID: messageID, Position: image.Position, Title: image.Title, URL: image.URL, ThumbnailURL: image.ThumbnailURL, Source: image.Source, Width: image.Width, Height: image.Height})
		}
		if err := s.messages.CreateImages(rows); err != nil {
			logrus.WithError(err).Warn("image results persist failed; continuing without images")
		}
	}
	return items, promptSources, images, nil
}

// buildMapData builds the map center (the user's location with its
// reverse-geocoded label) and geocodes the top source titles into nearby
// markers. Best-effort: geocoding failures and out-of-range results are
// dropped, and a total failure still yields the center so the frontend can
// render a map of the user's area. The radius (100 km) keeps "hospital near
// me" markers plausibly near the user instead of geocoding a listicle title
// to a faraway city.
func (s *AIService) buildMapData(ctx context.Context, location GeoLocation, placeLabel string, sources []SourceItem) (MapPoint, []MapPoint) {
	center := MapPoint{Label: placeLabel, Latitude: location.Latitude, Longitude: location.Longitude}
	if s.geo == nil || len(sources) == 0 {
		return center, nil
	}
	const (
		maxGeocode   = 5
		maxMarkers   = 6
		maxRadiusKM  = 100.0
		geocodeLimit = 1
	)
	titles := make([]string, 0, maxGeocode)
	for _, source := range sources {
		titles = append(titles, source.Title)
		if len(titles) == maxGeocode {
			break
		}
	}
	gctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	type geocodeResult struct {
		point GeocodePoint
		err   error
	}
	results := make([]geocodeResult, len(titles))
	var wg sync.WaitGroup
	for index, title := range titles {
		wg.Add(1)
		go func(index int, title string) {
			defer wg.Done()
			points, err := s.geo.Geocode(gctx, title, geocodeLimit)
			if err != nil {
				results[index] = geocodeResult{err: err}
				return
			}
			if len(points) > 0 {
				results[index] = geocodeResult{point: points[0]}
			}
		}(index, title)
	}
	wg.Wait()

	markers := make([]MapPoint, 0, maxMarkers)
	seen := make(map[string]bool, maxMarkers)
	for _, result := range results {
		if result.err != nil {
			continue
		}
		point := result.point
		if haversineKM(location, GeoLocation{Latitude: point.Latitude, Longitude: point.Longitude}) > maxRadiusKM {
			continue
		}
		key := fmt.Sprintf("%.3f,%.3f", point.Latitude, point.Longitude)
		if seen[key] {
			continue
		}
		seen[key] = true
		markers = append(markers, MapPoint{Label: point.Label, Latitude: point.Latitude, Longitude: point.Longitude})
		if len(markers) == maxMarkers {
			break
		}
	}
	return center, markers
}

// buildSearchSummary composes a non-AI, extractive summary from the top
// SearXNG snippets — no LLM tokens, no deep-read. It reads like a digest of
// what the top results say, with citation numbers pointing at the source list.
func buildSearchSummary(items []SourceItem) string {
	parts := make([]string, 0, 3)
	cites := make([]string, 0, 3)
	for _, item := range items {
		snippet := strings.TrimSpace(item.Snippet)
		if snippet == "" {
			continue
		}
		parts = append(parts, trimRunes(snippet, 180))
		cites = append(cites, fmt.Sprintf("[%d]", item.Position))
		if len(parts) == 3 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Here's what the top results say: " + strings.Join(parts, " ") + " " + strings.Join(cites, "")
}

// trimRunes truncates a string to at most n runes (unicode-safe).
func trimRunes(value string, n int) string {
	runes := []rune(value)
	if len(runes) <= n {
		return value
	}
	return string(runes[:n])
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
