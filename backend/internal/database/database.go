package database

import (
	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	if cfg.DBDriver == "sqlite" {
		if err := os.MkdirAll(filepath.Dir(cfg.DBSQLitePath), 0o750); err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(cfg.DBSQLitePath), &gorm.Config{})
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// seedSingleton inserts a singleton row only when the primary key is missing.
// Unlike FirstOrCreate, the lookup conditions on the primary key alone, so an
// admin-edited row (branding, queue knobs) is never re-inserted on boot.
func seedSingleton(db *gorm.DB, id uint, seed any) error {
	if err := db.First(seed, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.Create(seed).Error
		}
		return err
	}
	return nil
}

func MigrateAndSeed(db *gorm.DB, cfg config.Config) error {
	if err := db.AutoMigrate(&entities.User{}, &entities.SiteSettings{}, &entities.AIQueueConfig{}, &entities.AIProvider{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}, &entities.ImageResult{}, &entities.UsageLog{}, &entities.CrawlJob{}, &entities.SearchHistory{}, &entities.AnonymousUsage{}, &entities.Note{}, &entities.MapPoint{}, &entities.RegisterVisit{}, &entities.MiniApp{}, &entities.MiniAppApiDoc{}); err != nil {
		return err
	}
	if err := seedMiniAppApiDocs(db); err != nil {
		return err
	}
	if err := seedSingleton(db, 1, &entities.SiteSettings{ID: 1, SiteName: "Intellisearch"}); err != nil {
		return err
	}
	if err := seedSingleton(db, 1, &entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, SuggestionCacheHours: 6, DefaultDailyQuota: 3, MaxImageResults: 20, SessionTTLHours: 168}); err != nil {
		return err
	}
	// The seed-managed "local-ollama" provider mirrors the runtime env config
// (OLLAAMA_BASE_URL / AI_MODEL / AI_PROVIDER), which is the source of truth for
// host-run development. Insert it when missing and refresh its live fields on
// every boot so a changed .env no longer leaves a stale row behind.
	provider := &entities.AIProvider{Name: "local-ollama"}
	if err := db.Where("name = ?", provider.Name).First(provider).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&entities.AIProvider{ID: uuid.New(), Name: provider.Name, ProviderType: cfg.AIProvider, BaseURL: cfg.OllamaBaseURL, Model: cfg.AIModel, IsActive: true}).Error; err != nil {
			return err
		}
	} else if err := db.Model(provider).Updates(map[string]any{"provider_type": cfg.AIProvider, "base_url": cfg.OllamaBaseURL, "model": cfg.AIModel}).Error; err != nil {
		return err
	}
	if cfg.SuperOwnerEmail == "" || cfg.SuperOwnerPassword == "" {
		return nil
	}
	var count int64
	if err := db.Model(&entities.User{}).Where("role = ?", entities.RoleSuperOwner).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SuperOwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.Create(&entities.User{ID: uuid.New(), Name: "Super Owner", Email: cfg.SuperOwnerEmail, PasswordHash: string(hash), Role: entities.RoleSuperOwner, Status: entities.StatusActive}).Error
}

// seedMiniAppApiDocs records the Mini Apps platform API reference in the
// database (idempotent: only seeds when the table is empty). The docs drive
// the Studio's API list and the downloadable AI-API markdown export — they are
// stored data, not frontend code, so they never need a deploy to update.
func seedMiniAppApiDocs(db *gorm.DB) error {
	var count int64
	if err := db.Model(&entities.MiniAppApiDoc{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	// md joins lines so the markdown reads naturally while staying regular
	// (escapable) Go string literals.
	md := func(lines ...string) string { return strings.Join(lines, "\n") }
	docs := []entities.MiniAppApiDoc{
		{
			Section: "Overview", Title: "Mini Apps API", SortOrder: 1,
			Markdown: md(
				`Your mini app runs inside a sandboxed iframe that stays on this site's origin, so its JavaScript can call the same API the web app uses — including the AI question pipeline and every account endpoint — using the signed-in user's session.`,
				``,
				`## Conventions`,
				``,
				`- Base URL: /api/v1 (same origin, no CORS needed).`,
				`- Every response uses the envelope { "data", "errorCode", "errorMessage" }; success has empty errorCode/errorMessage.`,
				`- Authenticated calls send "Authorization: Bearer <token>". Because the iframe is same-origin, the session token is shared via localStorage — read it with localStorage.getItem('token').`,
				`- If the user is signed out, only public endpoints are reachable; AI asks are limited to anonymous guests.`,
				``,
				`A minimal fetch helper:`,
				``,
				"```js",
				`async function api(path, options = {}) {`,
				`  const token = localStorage.getItem('token')`,
				"  const header = token ? { Authorization: `Bearer ${token}` } : {}",
				`  const res = await fetch('/api/v1' + path, { ...options, headers: { 'Content-Type': 'application/json', ...header, ...(options.headers || {}) } })`,
				`  return res.json()`,
				`}`,
				"```",
			),
		},
		{
			Section: "AI (Ask)", Title: "POST /api/v1/ask", Method: "POST", Path: "/api/v1/ask", SortOrder: 10,
			Markdown: md(
				`Ask the platform AI (SearXNG search + crawler + LLM synthesis). It is powered by the AI handler, so it enforces the shared worker-pool queue, the owner-configured concurrency, the per-user rate limit, and the user's daily question quota — all without any extra wiring in your app.`,
				``,
				`Body: { "query": string, "mode": "enhanced" | "search" } (search returns raw web results with an extractive summary and costs zero AI tokens). You can pass "sessionId" to continue a conversation.`,
				``,
				"```js",
				`const { data } = await api('/ask', { method: 'POST', body: JSON.stringify({ query: 'cheapest laptop under 500', mode: 'enhanced' }) })`,
				`console.log(data.answer, data.sources)`,
				"```",
				``,
				`Errors: AISY02001 queue full (busy — try again), AISY02002 rate limited, AISY02003 daily quota exceeded, AISY02004 anonymous guest allowance used (sign in to continue), AISY01xxx provider unavailable/timeout/error.`,
			),
		},
		{
			Section: "AI (Ask)", Title: "URL submissions", Method: "POST", Path: "/api/v1/ask/url", SortOrder: 11,
			Markdown: md(
				`Ask the AI about a specific page. Body: { "url": string }. URL submissions are rate-limited much more strictly (5/hour) and are SSRF-guarded (internal/private addresses are blocked with AISY03002). Example:`,
				``,
				"```js",
				`const { data } = await api('/ask/url', { method: 'POST', body: JSON.stringify({ url: 'https://example.com/specs' }) })`,
				"```",
			),
		},
		{
			Section: "Mini Apps (CRUD)", Title: "List my apps", Method: "GET", Path: "/api/v1/me/mini-apps", SortOrder: 20,
			Markdown: `Returns the signed-in user's apps (full source): { "items": [MiniApp] } where MiniApp is { id, userId, name, slug, description, icon, html, css, js, visibility, isActive, createdAt, updatedAt }.`,
		},
		{
			Section: "Mini Apps (CRUD)", Title: "Create an app", Method: "POST", Path: "/api/v1/me/mini-apps", SortOrder: 21,
			Markdown: `Create a mini app. Body: { "name", "description"?, "icon"?, "html", "css"?, "js"?, "visibility": "public"|"private" }. Name is required (≤ 80 chars) and must be unique enough to form a free slug (collision → MINI01005); each source field is capped (html/css ≤ 60k runes each, js ≤ 120k runes). Returns the created app.`,
		},
		{
			Section: "Mini Apps (CRUD)", Title: "Get / update / delete my app", Method: "PATCH", Path: "/api/v1/me/mini-apps/:id", SortOrder: 22,
			Markdown: md(
				`- GET /api/v1/me/mini-apps/:id — fetch one of your apps (owner only; MINI01003 when missing).`,
				`- PATCH /api/v1/me/mini-apps/:id — partial update of any field above (blank name keeps the current one).`,
				`- DELETE /api/v1/me/mini-apps/:id — delete it (owner only).`,
				``,
				`App ids are UUIDs.`,
			),
		},
		{
			Section: "Mini Apps (CRUD)", Title: "Public apps", Method: "GET", Path: "/api/v1/mini-apps", SortOrder: 23,
			Markdown: md(
				`List every active public app — { "items": [{ id, userId, name, slug, description, icon, visibility, isActive, updatedAt }] } (no source). Use this to render an app launcher/gallery.`,
				``,
				`Open and run a single app by slug:`,
				``,
				"```js",
				`const { data } = await api('/mini-apps/' + slug) // full MiniApp source`,
				"```",
				``,
				`Private apps return MINI03001 unless the request is from the owning user.`,
			),
		},
		{
			Section: "Account", Title: "Who am I", Method: "GET", Path: "/api/v1/me", SortOrder: 30,
			Markdown: md(
				`Returns the signed-in user's profile plus { daily usage, remaining quota }:`,
				``,
				"```js",
				`const { data } = await api('/me')`,
				`console.log(data.name, data.email, data.usage) // { usedToday, quota, remaining }`,
				"```",
				``,
				`PATCH /api/v1/me updates name/email; POST /api/v1/me/avatar uploads an avatar.`,
			),
		},
	}
	return db.Create(&docs).Error
}
