# Implementation Status

**As of:** 2026-08-17
**Overall status:** Early MVP implementation; M2–M4 (public UI, AI pipeline, Owner Control Panel) are now delivered. M5 hardening items remain.

This document is the implementation truth for the repository. The PRD and SDD remain the target product/design specification; a checked item below means the behavior exists in code, not merely in planning documentation.

## Delivered

### Frontend — public pages

- Vue 3, TypeScript, Vite, Vue Router, and Pinia bootstrapping.
- Public routes: `/`, `/search`, `/login`, and `/account`.
- Clean/premium visual revamp for the public pages, with responsive layouts and light/dark/system theme tokens.
- Reusable `AppHeader`, `AskBox`, `UrlAskBox`, `ErrorBanner`, `BaseButton`, `Tabs`, `ThemeToggle`, `SourceCard`, and new shared form/modal/chart primitives.
- Main page simplified to a Google-like centered search entry point.
- **Result page wired to the real AI API:** asks the query on load, renders the cited answer + numbered sources, follow-up thread (same `sessionId`), skeleton loading, friendly error banner with retry, and a URL-read box in the no-sources state.
- **Account settings wired to the API:** sign-in-gated profile view (name/email edit, avatar upload with initials fallback), daily AI usage meter with quota, session tab with logout and (for Super Owners) a control-panel shortcut.
- **Unified sign-in page (`/login`):** one glass-themed auth page for every account type — Super Owners sign in here too (the old `/admin/login` redirects here) and reach the control panel via their account menu. After signing in everyone lands on the **main page** (previously `/account`).
- **Google SSO (optional):** `GET /api/v1/auth/google` starts the OAuth authorization-code flow (CSRF `state` in an httpOnly cookie); the callback exchanges the code, finds-or-creates the user, issues the app JWT, and redirects to `/auth/callback?token=…` which completes the session and lands on the main page. New Google accounts become `general_user` with no usable password; disabled when `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` are unset (`AUTH01003`).

### Frontend — Owner Control Panel (SDD M4)

- **Registration:** the same auth page adds a **Create account** tab (`/login?mode=register`, `/register` redirects there) — name + email + password (≥ 8 chars) creates a `general_user` via `POST /api/v1/auth/register` (`AUTH01004` invalid data, `AUTH01005` duplicate email) and lands the user signed in. **Google SSO registers too**: the OAuth callback find-or-creates the account, so "Sign up with Google" works with zero extra backend setup.
- **Authenticated admin shell** (`ControlPanel`): persistent sidebar on desktop (≥1024 px) with a **parent → child accordion** (expandable/collapsible groups, active parent auto-expanded, active child highlighted); below 1024 px it collapses into a **hamburger-triggered slide-in drawer with a backdrop overlay**, the current page name is shown in the header, and selecting a page navigates and closes the drawer.
- **User management** (`/admin/users`, `/admin/users/suspended`): debounced search, pagination, create/edit modal (name, email, password on create, role, daily quota), suspend/reinstate/delete with confirmation modals, live refresh on focus.
- **Statistics** (`/admin/stats`, `/admin/stats/top`, `/admin/stats/usage`, `/admin/stats/ai`): overview stat cards + **daily (7-day) and weekly (8-week) trend bar charts**, top-query and heaviest-user charts, full top-queries list, per-user usage table with usage bars, and an AI-service page (success/failure rate, per-provider performance, **error-code grouping with a filter-by-type dropdown**, latency percentiles, queue health).
- **AI providers** (`/admin/ai/providers`): create/edit/delete provider cards with type (Ollama / OpenAI-compatible / **Pollinations.ai** / **Hugging Face**), auto-filled endpoint + model presets per type, base URL, model, JSON model parameters, encrypted API key (blank keeps existing; hidden for keyless providers), and single-click "set active" routing. **Ollama providers get a live server inspector**: a "Load models & health" button (auto-loads on open) fetches the model list into a picker (`GET /api/tags`) plus a health card with the server version and loaded-model CPU/GPU/memory stats (`GET /api/version` + `/api/ps`) — all proxied through the Go API (`ADMN06001`), never direct from the browser.
- **Queue & limits** (`/admin/ai/queue`): edit `max_concurrent`, `max_queue_size`, `request_timeout_ms`, `per_user_rate_limit` with immediate apply — no redeploy.
- **Branding** (`/admin/branding/identity`, `/admin/branding/logo`): site name/tagline editing with live preview, logo upload with drag & drop, and **logo removal (delete → falls back to the default initials mark)**; all apply to the public main page immediately (site store refresh).
- Frontend auth store (JWT in localStorage, `restore()` on boot, 401 expiry handling), admin router guards, typed API client with friendly per-error-code messages, and a toast notification system reused across the panel.
- **Session persistence on hard refresh:** the router guard restores the user whenever a token exists but the in-memory profile is missing, so a reload no longer appears to log the user out.
- **Login UX (glass theme):** the auth page was fully revamped — animated aurora blobs, a frosted-glass card with backdrop blur, translucent glass inputs and buttons, and a segmented Sign in / Create account switcher with slide transitions, success checkmark animation, and error banners (including Google OAuth failures via `?error=`). All animations respect `prefers-reduced-motion`.
- **Search history:** every non-URL ask by a signed-in user is recorded in a dedicated `search_history` table (separate from `usage_logs` so admin stats/quotas are unaffected). The main page shows the user's **recent searches** as clickable chips under the ask box, plus **AI-composed suggestions** derived from their history (composed by the active LLM, cached ~10 min, refreshable via a ↻ button, gracefully hidden when the provider is down or history is empty). **Account Settings** lists the full history with timestamps and a **Clear history** button (with confirmation). Endpoints: `GET/PATCH/DELETE /api/v1/me/history` + `GET /api/v1/me/history/suggestions`; error codes `USER03001/2`.

### Backend foundation

- Go/Gin API with the common response envelope and typed error-code catalog (`<FEATURE><SUBSET><ERROR>` format enforced by a unit test over `contracts/errors.go`).
- Logrus startup/recovery logging and configured CORS middleware.
- GORM AutoMigrate and idempotent seeds for users, site settings, AI queue configuration, providers, chat sessions, messages, search results, usage logs, and crawl jobs. Singleton seeds (`site_settings`, `ai_queue_configs`) look up by primary key only, so admin-edited rows (custom branding, tuned queue knobs) survive reboots instead of being re-inserted.
- Database selection by environment: local host-run defaults to SQLite; production and Docker API services explicitly use PostgreSQL.
- Site branding endpoint reads persisted `site_settings` rather than hardcoded response data.
- JWT login, authenticated profile read/update, avatar upload, and Super Owner role middleware.
- Admin endpoints for user CRUD, statistics, provider CRUD, queue config, site settings, and logo upload (each route requires Super Owner); logo deletion returns branding to the default mark.
- `GET /api/v1/admin/stats/trends` returns daily/weekly question counts (bucketed in the service so SQLite and PostgreSQL behave identically).
- SearXNG JSON client with source/domain mapping.
- Go URL validation and crawler-side URL validation for SSRF protection.

### AI pipeline (SDD M3)

- **Single AI handler** (`internal/handlers/ai_handler.go`) as the only entry point for AI work, owning a **worker pool + bounded queue** whose knobs (`max_concurrent`, `max_queue_size`, `request_timeout_ms`, `per_user_rate_limit`) are read from `ai_queue_configs` with a 5-second cache — admin edits take effect without redeploying (verified live: DB edit applied within ~5s without restart).
- **`POST /api/v1/ask`** and **`POST /api/v1/ask/url`** implemented end to end: session/message/usage-log persistence, SearXNG search → source cards, deep-read of top N pages through the crawler, LLM synthesis with inline citations, and typed error mapping with sanitized client messages.
- **LLM provider abstraction** (`internal/services/llm_service.go`): Ollama (`/api/chat`) plus OpenAI-compatible clients — OpenAI-compatible (`/v1/chat/completions`), **Pollinations.ai** (`/v1/chat/completions`, Bearer key required, base `https://gen.pollinations.ai`), and **Hugging Face** (`/chat/completions`, Bearer key); provider API keys decrypted at call time from AES-GCM storage.
- **CrawlService** (`internal/services/crawl_service.go`): SSRF-guarded URL validation, `crawl_jobs` lifecycle (queued → running → completed/failed/blocked), best-effort parallel deep-read of the top sources.
- **Redis sliding-window rate limiter** (per-user or per-IP; URL submission capped at 5/hour) and **per-user daily quota** (`ai_daily_quota`, 0 = unlimited) enforced before enqueue.
- **`GET /api/v1/sessions/:id`**: owner-only session retrieval with message history and per-message `search_results`.
- Runtime queue metrics (`queueDepth`, `inFlight`, `rejected`, `maxConcurrent`) exposed on the handler for the admin AI-stats page.
- Graceful degradation: SearXNG down → the LLM still answers without web sources.
- Error codes: `AISY01001/2/3/4`, `AISY02001/2/3`, `AISY03002/3/4`, `SESS01001/2`, plus `AUTH*`, `USER*`, `SITE*`, and `ADMN*`.

### Runtime and documentation

- Local script provisions Redis, SearXNG (host 8081 → container 8080), and the Playwright crawler (host 3001 → container 3002, image built from `crawler/`) via Podman and uses `backend/data/intellisearch.db` for SQLite; `SEARXNG_BASE_URL=http://localhost:8081` and `CRAWLER_BASE_URL=http://localhost:3001` are exported for the host-run backend (Docker Compose overrides both to the internal service names).
- Host-exposed ports avoid the commonly occupied set: the API listens on **8088** (was 8080) and the production frontend on **8082** (was 80); SearXNG and the crawler stay internal-only inside the Docker network, so no host conflict. Postgres (5432) and Ollama (11434) are reused external containers, not deployed by this stack.
- Production Compose sets `DB_DRIVER=postgres`; the API service's `env_file` resolves `${ENV_FILE:-.env}` so `deploy-prod.sh` injects the gitignored `.env.production` (strong secrets) instead of the dev `.env`. The stack deploys only its own services — PostgreSQL and Ollama are reused: the API attaches to external `SHARED_DB_NETWORK` and `OLLAMA_NETWORK` networks (no DB/Ollama service in `docker-compose.prod.yml`). All three Dockerfiles build (`backend` on `golang:1.25` with `-p 2` compile parallelism for small hosts, `frontend` via `npm run build` → nginx, `crawler` via `npm ci` with its generated `package-lock.json`).
- API contract, UI revamp proposal/SDD, and this status document are maintained under `docs/`.

## In progress

- M1 request logging is covered by Gin's logger; Redis startup retries (30 s) so slower container port-forwards don't disable rate limiting.
- M5 hardening: hard rate limits on URL submission and ask are in place; remaining items are QR payment-code signing (per PRD/SDD/Sprint 5) and suggested-follow-up generation (chips).

## Not implemented yet

- Suggested follow-up generation (chips) on the result page.
- QR payment-code generation/HMAC signing (Sprint 5 plan).
- Production end-to-end deployment verification with a live Ollama + SearXNG + crawler stack.

## Verification status

- `go test ./...`: passing (envelope + error-code format, AES-GCM round-trip, Ollama/OpenAI client mapping, SearXNG mapping, SSRF guard, crawl lifecycle, AI pipeline persistence + graceful degradation, queue overflow → `AISY02001`, rate limit → `AISY02002`, daily quota → `AISY02003`, logo upload/delete with default fallback, stats trends daily/weekly bucketing, search-history record/clear/suggestions + router coverage, register success/duplicate/weak-password + router coverage, Ollama models/health parsing + invalid-URL rejection + admin router coverage). Race detector clean.
- `go build ./...`, `go vet ./...`: passing.
- `npm run typecheck` and `npm run build`: passing (all admin views compile; lazy-loaded routes).
- Live smoke: `/health`, `/site`, `/ask` (provider-down → 503 `AISY01001` with sanitized message), empty query → 400 `AISY01004`, blocked URL → 403 `AISY03002`, invalid URL → 400 `AISY03003`, sessions without auth → 401; Redis rate limiting + hot-reloaded queue config verified; headless-browser check of MainPage and ResultPage (query, error banner, retry, URL box) passing.
- Full Docker stack, live PostgreSQL, live Ollama, live SearXNG, crawler-in-Docker, and browser E2E with a real model: not yet verified.

## Next implementation order

1. Suggested follow-up chips on the result page (SDD M3 polish).
2. QR payment-code generation/signing and any remaining Sprint 5 hardening tasks.
3. Production end-to-end deployment verification with a live Ollama + SearXNG + crawler stack.