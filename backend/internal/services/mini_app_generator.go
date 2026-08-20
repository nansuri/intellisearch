package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/models/entities"
)

// MiniAppDraft is the LLM-generated source of a mini app, returned by the AI
// generation job and persisted by MiniAppService.CreateDraft.
type MiniAppDraft struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	HTML        string `json:"html"`
	CSS         string `json:"css"`
	JS          string `json:"js"`
}

const maxGeneratePromptLength = 2000

// GenerateMiniApp composes a complete mini app (HTML + CSS + JS) from a single
// natural-language prompt. It runs through the same LLM service as asks and is
// gated by GenerateMiniApp on the handler side, so it inherits the AI handler's
// queue, rate limit, and the user's daily question quota. Every call is
// recorded in usage_logs attributed to the user (like an ask) so generation
// counts toward the quota.
func (s *AIService) GenerateMiniApp(ctx context.Context, userID uuid.UUID, prompt string) (MiniAppDraft, error) {
	clean := SanitizeQuery(prompt)
	if clean == "" {
		return MiniAppDraft{}, ErrInvalidQuery
	}
	started := time.Now().UTC()
	provider, _ := s.providers.Active()
	usageLog := entities.UsageLog{ID: uint64(time.Now().UnixNano()), UserID: &userID, Query: "generate mini app: " + trimRunes(clean, 100), Status: entities.MessageStatusQueued, CreatedAt: started}

	fail := func(cause error) (MiniAppDraft, error) {
		code, message := CodeForError(cause), SanitizedErrorMessage(cause)
		usageLog.LatencyMS = int(time.Since(started).Milliseconds())
		usageLog.Status = entities.MessageStatusFailed
		usageLog.ErrorCode = &code
		usageLog.ErrorMessage = &message
		_ = s.usageLogs.Create(&usageLog)
		return MiniAppDraft{}, cause
	}

	system := `You are a front-end builder for a platform that renders user-made mini apps inside a sandboxed iframe. Turn the user's requirement into a single self-contained mini app and return ONLY valid JSON with this exact shape:
{
  "name": "short, human-readable app name (max 60 chars)",
  "description": "one-line description (max 200 chars)",
  "icon": "a single emoji",
  "html": "the app's markup as a string",
  "css": "the app's styles as a string (no <style> tag)",
  "js": "the app's behavior as a string (no <script> tag)"
}
Rules:
- The html string is the full body content (no <html>, <head>, or <body> tags).
- The css string is plain CSS scoped to your app.
- The js string is plain JavaScript executed after the elements exist; no imports, no frameworks, no external CDNs. Prefer const/function with let-free style if possible.
- Make the app look polished and modern: use system fonts, subtle shadows, rounded corners, and a clean color palette. Keep it responsive for narrow mobile iframes (use max-width and %/flex layouts).
- Do not access document.cookie or parent windows; the sandbox restricts them.
- Escape any backslashes and quotes properly so the output is parseable JSON.`
	generated, err := s.llm.Generate(ctx, system, []ChatMessage{{Role: "user", Content: clean}}, GenerateOptions{Temperature: 0.6, MaxTokens: 4000})
	if err != nil {
		return fail(err)
	}
	draft, err := parseMiniAppDraft(generated.Content)
	if err != nil {
		logrus.WithError(err).WithField("prompt", clean).Warn("mini app generation returned unparseable output")
		return fail(ErrMiniAppGenerate)
	}
	if err := s.validateDraft(&draft); err != nil {
		logrus.WithError(err).Warn("mini app draft failed validation; returning parse error")
		return fail(ErrMiniAppGenerate)
	}
	usageLog.LatencyMS = int(time.Since(started).Milliseconds())
	usageLog.Status = entities.MessageStatusCompleted
	usageLog.InputTokens = generated.InputTokens
	usageLog.OutputTokens = generated.OutputTokens
	usageLog.GenerateMS = int(generated.Duration.Milliseconds())
	if provider.ID != uuid.Nil {
		usageLog.ProviderID = &provider.ID
	}
	_ = s.usageLogs.Create(&usageLog)
	return draft, nil
}

// validateDraft enforces the same caps as MiniAppService before persisting so a
// misbehaving model cannot create an oversized row.
func (s *AIService) validateDraft(draft *MiniAppDraft) error {
	name := strings.TrimSpace(draft.Name)
	fill := func(ptr *string, fallback string) string {
		out := strings.TrimSpace(*ptr)
		if out == "" {
			out = fallback
		}
		return out
	}
	draft.Name = trimRunes(name, maxMiniAppNameLength)
	if draft.Name == "" {
		draft.Name = "AI mini app"
	}
	draft.Description = trimRunes(draft.Description, maxMiniAppDescriptionLength)
	draft.Icon = trimRunes(draft.Icon, maxMiniAppIconLength)
	draft.HTML = fill(&draft.HTML, "<h1>Hello</h1>")
	draft.CSS = trimRunes(draft.CSS, maxMiniAppSourceLength)
	draft.JS = trimRunes(draft.JS, maxMiniAppJSLength)
	return nil
}

// parseMiniAppDraft extracts a MiniAppDraft from an LLM reply, tolerating
// markdown fences around the JSON object.
func parseMiniAppDraft(raw string) (MiniAppDraft, error) {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	if idx := strings.LastIndex(text, "```"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}
	var draft MiniAppDraft
	if err := json.Unmarshal([]byte(text), &draft); err != nil {
		return MiniAppDraft{}, err
	}
	return draft, nil
}