# Implementation Guide — Intellisearch MVP

> **Read this file top to bottom before writing any code.** It is the single source of truth for implementing the single-shot MVP. It is self-contained: do not block on the other docs. If a detail is missing here, fall back to the [PRD](docs/PRD/index.md) → [SDD](docs/SDD/index.md) → [Sprint Plan](docs/tech_stack/sprints/MVP_Sprint_Plan.md), in that order, and update this file if you discover a gap.
>
> `AGENTS.md` at the repo root contains the hard rules — this file restates them and adds the concrete decisions you need. **Both apply.**

> **Implementation note (2026-08-17):** This guide describes the complete target MVP. The current delivery state is tracked in [`docs/tech_stack/IMPLEMENTATION_STATUS.md`](docs/tech_stack/IMPLEMENTATION_STATUS.md). Local host-run development uses SQLite; production uses PostgreSQL.

---

## 0. What you are building

A Google-like AI search main page (white-label, no hardcoded branding):

- **Main page** (`/`): centered ask box + "Ask Me" button, header with site name/logo from `site_settings`, top-right avatar button. Submitting navigates to the Result Page.
- **Result Page** (`/search?q=…`): search-results-style page — query header ("About N sources · X.Xs"), cited **AI Summary card**, numbered **Web sources list** (from SearXNG), **follow-up thread** with **suggested follow-up chips**, and a **persistent ask box**.
- **Account Settings** (`/account`): **tabbed** sub-menus — Profile (avatar upload), AI Limit & Usage (usage bar + quota), Session (logout).
- **Owner Control Panel** (`/admin`): login-gated, **sidebar parent → child menu** (accordion on desktop, hamburger drawer on mobile), full **CRUD** on management menus via **reusable Edit + Delete-confirmation modals**, user management, statistics (incl. **detailed AI stats**: success rate, error list, latency percentiles, queue health), AI provider/queue settings, branding settings.
- **Dark mode** app-wide (design tokens, Light/Dark/System toggle), **responsive/mobile-first** (≥360px).

Backend: **Go + Gin + GORM + Logrus**, DDD layering, single AI handler with worker pool + queue, **SearXNG** (self-hosted metasearch, JSON API, backend-to-backend), **Playwright crawler sidecar** (Node), SQLite for host-local development, and PostgreSQL + Redis for Docker/production.

---

## 1. Non-negotiable rules

1. **No `.vue` file exceeds 1000 lines.** Break features into smaller components under `src/domains/{feature}/components/`.
2. **Reusable components only.** Shared UI (buttons, inputs, modals, tabs, badges, cards, status pills) is extracted into reusable components and used everywhere — never duplicated.
3. **Backend DDD layering:** `handlers/` → `services/` → `repositories/` → `models/entities/`. One specialized **AI handler** is the only entry point for all AI work.
4. **Response envelope everywhere:** every backend response is `{ "data": ..., "errorCode": "...", "errorMessage": "..." }` (success = empty `errorCode`/`errorMessage`).
5. **Typed error codes:** no raw strings at call sites. All codes are constants defined in `backend/internal/contracts` with a comment each, in format `<FEATURE><SUBSET><ERROR>` (4 uppercase letters, 2 digits, 3 digits). See §4.2 catalog.
6. **Logrus logging:** every backend error is logged with its internal cause + structured fields; the frontend only ever receives sanitized `errorMessage`. Never log secrets/credentials.
7. **No hardcoded branding:** site name, logo, tagline render from `site_settings` (seeded with a generic default). No hardcoded brand strings in UI.
8. **No hardcoded colors:** all UI colors come from **design tokens** (CSS custom properties) with light + dark variants.
9. **Secrets only in env:** nothing secret in code or committed files. API keys encrypted at rest (AES-GCM, key from env).
10. **Browser never calls Ollama/SearXNG directly** — only the Go API. Validate CORS.
11. **SSRF guard:** the crawler must never reach internal services; URL submissions are validated (scheme/host) and rate-limited.
12. **Frontend styles are modular:** separate files for tokens, base, shell/layout, forms, components, features — no monolithic CSS.

---

## 2. Target repository layout

```
.
├── AGENTS.md
├── README.md
├── implementation.md            ← this file
├── .env.example                 ← all env vars documented here
├── .gitignore                   ← ignores .env, node_modules, dist, uploads/, *.log
├── run-local.sh                 ← docker compose up (dev)
├── deploy-prod.sh               ← production build + health-check gate
├── docker-compose.yml           ← dev stack
├── docker-compose.prod.yml      ← prod stack
├── backend/                     ← Go API
│   ├── Dockerfile
│   ├── go.mod / go.sum
│   └── internal/
│       ├── contracts/           ← envelope helpers + typed error-code constants
│       ├── config/              ← env loading
│       ├── models/              ← GORM entities
│       ├── repositories/        ← DB access per entity
│       ├── services/            ← auth, user, site, stats, ai (llm/search/crawl)
│       ├── handlers/            ← HTTP handlers incl. ai handler (worker pool)
│       ├── middleware/          ← cors, logging, recovery, auth, rbac, ratelimit
│       └── router/              ← route registration
│   └── cmd/server/main.go
├── crawler/                     ← Node + Playwright sidecar (page text fetcher)
│   ├── Dockerfile
│   ├── package.json
│   └── src/index.js             ← POST /fetch { url } → { title, text }
├── frontend/                    ← Vue 3 + TS + Vite
│   ├── Dockerfile               ← multi-stage → nginx (serves dist/, proxies /api)
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.ts
│       ├── App.vue
│       ├── router/index.ts
│       ├── services/api.ts      ← typed client + envelope parsing
│       ├── stores/              ← auth.ts, site.ts, settings.ts, admin.ts, theme.ts
│       ├── styles/              ← tokens.css (light/dark), base.css, layout.css, forms.css, components.css, features.css
│       ├── components/          ← app-wide reusable components
│       └── domains/
│           ├── ask/             ← components: AskBox, ResultPage pieces, SourceCard…
│           ├── account/         ← tabs, profile, usage, session
│           └── admin/           ← sidebar menu, stats, users, ai settings, branding
└── docs/                        ← planning docs (already present, do not delete)
```

---

## 3. Environment, config, scripts

Root `.env` is the **single source** of local env values (backend, frontend, docker). Frontend vars use the `VITE_` prefix. Create `.env.example` with **every** var and a comment.

```env
# App
APP_ENV=development
PORT=8080
CORS_ORIGINS=http://localhost:5173

# PostgreSQL
DB_DRIVER=sqlite                         # sqlite for host-local; postgres for production/Docker
DB_SQLITE_PATH=./backend/data/intellisearch.db
DB_HOST=postgres
DB_PORT=5432
DB_USER=aimain
DB_PASSWORD=aimain
DB_NAME=aimain
DB_SSLMODE=disable

# Redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=

# Auth & secrets
JWT_SECRET=change-me-32-chars-min
JWT_TTL_HOURS=24
ENCRYPTION_KEY=32-byte-aes-gcm-key
SUPER_OWNER_EMAIL=owner@example.com      # seeded on first boot
SUPER_OWNER_PASSWORD=change-me

# AI
AI_PROVIDER=ollama                        # ollama | openai_compatible (default provider type seeded)
OLLAMA_BASE_URL=http://ollama:11434
OPENAI_BASE_URL=
OPENAI_API_KEY=
AI_MODEL=llama3.2

# Search & crawler
SEARXNG_BASE_URL=http://searxng:8080
SEARXNG_TIMEOUT_MS=10000
CRAWLER_BASE_URL=http://crawler:3000
CRAWLER_TIMEOUT_MS=15000
CRAWL_TOP_N=3                              # top N source pages deep-read per query

# Frontend (used by Vite build)
VITE_API_BASE_URL=/api/v1                  # same-origin via nginx proxy in prod; http://localhost:8080/api/v1 in dev
```

- `run-local.sh`: starts Redis with Podman, uses SQLite for the host-run API, and prints the URLs (frontend :5173, API :8080/health).
- `deploy-prod.sh`: `docker compose -f docker-compose.prod.yml up -d --build`, then poll `GET /health` until it returns `{ "data": { "status": "ok" } }`; fail loudly otherwise.
- Both scripts read the root `.env` (docker compose does automatically for `${VAR}` interpolation).

---

## 4. Backend (Go)

Module name: `intellisearch` (adjustable). Go 1.22+. Dependencies: `gin-gonic/gin`, `gorm.io/gorm`, `gorm.io/driver/postgres`, `sirupsen/logrus`, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `github.com/redis/go-redis/v9`.

### 4.1 Bootstrap (`cmd/server/main.go`)

1. Load config from env (`internal/config`).
2. Init Logrus (JSON formatter; level from `APP_ENV`).
3. Connect GORM → the configured SQLite or PostgreSQL driver; run `AutoMigrate` for every entity in §4.4; seed singleton rows + super owner (§4.4.3).
4. Connect Redis.
5. Build repositories, services, the AI worker pool (§4.7), handlers, router; start `:8080` (graceful shutdown on SIGTERM/SIGINT).

### 4.2 Contracts (`internal/contracts`)

- `envelope.go` — `type Envelope struct { Data any `json:"data"`; ErrorCode string `json:"errorCode"`; ErrorMessage string `json:"errorMessage"` }` plus helpers `OK(data)`, `Fail(code, msg)`.
- `errors.go` — typed constants, one const per code, each with a `//` comment. **Error-code catalog** (extend as needed, keep the format):

| Code | Meaning |
| --- | --- |
| `AUTH01001` | invalid credentials |
| `AUTH01002` | session expired / invalid token |
| `AUTH02001` | forbidden — super_owner required |
| `USER01001` | user not found |
| `USER01002` | profile update validation failed |
| `USER02001` | avatar upload failed |
| `USER02002` | avatar rejected (type/size) |
| `AISY01001` | AI provider unavailable |
| `AISY01002` | AI provider timeout |
| `AISY01003` | AI provider returned an error |
| `AISY02001` | queue full — busy, try again |
| `AISY02002` | per-user rate limit exceeded |
| `AISY02003` | daily question quota exceeded |
| `AISY03001` | search backend (SearXNG) unavailable |
| `AISY03002` | URL rejected (SSRF guard / blocked) |
| `AISY03003` | invalid URL |
| `AISY03004` | crawl failed |
| `SESS01001` | session not found |
| `SESS01002` | session access denied |
| `ADMN01001` | user operation failed |
| `ADMN02001` | statistics computation failed |
| `ADMN03001` | provider not found |
| `ADMN03002` | invalid provider configuration |
| `ADMN04001` | queue config validation failed |
| `ADMN05001` | site settings validation failed |
| `ADMN05002` | logo upload failed |
| `SITE01001` | site settings not seeded |

### 4.3 Middleware (`internal/middleware`)

- **CORS**: allow only `CORS_ORIGINS` (comma-separated), `OPTIONS` preflight, headers `Content-Type, Authorization`.
- **RequestLogging**: Logrus with method, path, status, latency, user id if present.
- **Recovery**: recover panics → 500 envelope `AISY01001`-style generic message; log the stack.
- **Auth** (`RequireAuth`): validate `Authorization: Bearer <jwt>`; attach user to context; 401 `AUTH01002` on missing/expired.
- **RBAC** (`RequireRole("super_owner")`): 403 `AUTH02001` if role mismatch. Applied to **every** `/admin` route and `/auth/logout`.

### 4.4 Models, migrations, seeds (`internal/models`)

Use GORM structs with `uuid` PKs (`gorm.io/gorm` + a uuid type, e.g., `github.com/google/uuid`). AutoMigrate all. Enums as Go string types + DB `varchar` with CHECK constraints (or just varchar + validation in services).

Entities (fields exactly per SDD §7.1):

1. **User** — id, name, email (unique), password_hash, role (`general_user`|`super_owner`), status (`active`|`suspended`), created_at, last_login_at, avatar_url (nullable), ai_daily_quota (int, 0 = global default).
2. **ChatSession** — id, user_id FK, title (= first question), created_at/updated_at.
3. **Message** — id, session_id FK, role (`system`|`user`|`assistant`), content (text), status (`queued`|`streaming`|`completed`|`failed`), created_at.
4. **SearchResult** — id, message_id FK, position (int = citation number), title, url, domain, snippet, created_at.
5. **AIProvider** — id, name, provider_type (`ollama`|`openai_compatible`), base_url, model, parameters (jsonb: temperature, max_tokens, context window), api_key_encrypted (nullable text), is_active (bool, only one active), created_at/updated_at.
6. **AIQueueConfig** — singleton row: id=1, max_concurrent, max_queue_size, request_timeout_ms, per_user_rate_limit (0 = unlimited).
7. **SiteSettings** — singleton row: id=1, site_name, logo_url (nullable), tagline (nullable), updated_at.
8. **UsageLog** — id, user_id (nullable), query (sanitized), latency_ms, status, error_code (nullable), error_message (nullable, sanitized), provider_id (nullable FK), created_at.
9. **CrawlJob** — id, user_id FK, url, status (`queued`|`running`|`completed`|`failed`|`blocked`), created_at, finished_at.
10. **QRPaymentCode** — id, user_id FK, payload, expires_at, signature, created_at (security-scoped only; no checkout UI).

**Seeds** (idempotent — only if row missing):
- `AIQueueConfig` id=1: max_concurrent=4, max_queue_size=20, request_timeout_ms=60000, per_user_rate_limit=10.
- `SiteSettings` id=1: site_name="Intellisearch" (generic), tagline="" — **do not** brand it.
- `AIProvider` "local-ollama": type `ollama`, base_url=`OLLAMA_BASE_URL`, model=`AI_MODEL`, is_active=true.
- Super Owner user from `SUPER_OWNER_EMAIL`/`SUPER_OWNER_PASSWORD` (bcrypt hash) with role `super_owner`, status `active` — only if no super owner exists.

### 4.5 Repositories (`internal/repositories`)

One file per aggregate: `user_repo.go`, `session_repo.go`, `message_repo.go`, `search_result_repo.go`, `provider_repo.go`, `queue_config_repo.go`, `site_settings_repo.go`, `usage_log_repo.go`, `crawl_job_repo.go`, `qr_code_repo.go`. Standard GORM CRUD. Repository layer returns `(entity, error)`; no envelope/HTTP concerns here.

### 4.6 Services (`internal/services`)

- **AuthService** — `Login(email, password)` → verify bcrypt, update `last_login_at`, issue JWT (claims: user_id, role, exp from `JWT_TTL_HOURS`). `Logout` is client-side token discard (stateless JWT; keep endpoint for session UX).
- **UserService** — profile get/update, avatar upload (validate type `image/*` ≤ 2MB; save to `./uploads/avatars/{user_id}.{ext}`, return URL), daily usage + remaining quota for today (count `UsageLog` where status=completed, not error), suspend/block, CRUD for admin.
- **SiteService** — get/update `SiteSettings`; logo upload (same pattern as avatars, `./uploads/branding/logo{ext}`); public `GetPublicSite()` used by `/api/v1/site`.
- **StatsService** — user stats (questions per day/week, active users, top queries, per-user usage) and **AI stats** (§4.8 `/admin/stats/ai`): success/failure rate overall + per provider/model (from `UsageLog`), error list grouped by `error_code` with count + last seen (filter `?type=`), latency avg + p50/p95/p99 (Postgres percentile queries on `latency_ms` where status=completed), queue health (from AI handler runtime metrics).
- **AIService** — the AI pipeline (§4.7); internally uses:
  - **LLMService** — provider abstraction: `Generate(prompt, opts) (text, err)`. Implement `ollama` client (`POST {base}/api/chat`) and `openai_compatible` client (`POST {base}/v1/chat/completions`, Bearer `api_key` if set). Decrypt `api_key_encrypted` before use; never log it. Map timeouts → `AISY01002`, connection errors → `AISY01001`.
  - **SearchService** — `Search(query) ([]SearchResultItem, error)`: `GET {SEARXNG_BASE_URL}/search?q={query}&format=json`; map `results[]` (title, url, content→snippet) → items; derive `domain` from URL host. Timeout → `AISY03001`.
  - **CrawlService** — `FetchPage(url) (text, error)`: validate scheme (http/https only) + reject private/internal hosts (SSRF, §4.9), then `POST {CRAWLER_BASE_URL}/fetch {url}`. Mark `CrawlJob` completed/failed/blocked.

### 4.7 AI handler + worker pool (`internal/handlers/ai_handler.go`)

- One `AIHandler` struct owning: a **worker pool** (`max_concurrent` workers) + **bounded queue** (`max_queue_size`), both read from `AIQueueConfig` (reloaded from DB on every enqueue — a short TTL cache is fine — so the Owner Control Panel changes take effect without redeploy).
- `Enqueue(ctx, job)`:
  - Apply per-user rate limit (Redis sliding window, `per_user_rate_limit`/min) → else `AISY02002`.
  - Apply daily quota (UsageLog count today for user) → else `AISY02003`.
  - Queue full → reject `AISY02001` "We're busy — try again in a moment."
  - Worker: run job with `request_timeout_ms` budget; on timeout mark message + UsageLog `failed` with `AISY01002`.
- Job pipeline for `POST /api/v1/ask`:
  1. Create ChatSession (title = query) + user Message (queued → streaming → completed/failed) + UsageLog.
  2. `SearchService.Search(query)` → listings; persist `SearchResult` rows (positions 1..N).
  3. If listings exist: `CrawlService.FetchPage` for top `CRAWL_TOP_N` URLs (in parallel); concatenate page text (truncated, e.g. 8k chars each) + snippets.
  4. Build LLM prompt: system = "Answer with inline citations [n] matching the numbered sources; be concise."; user = query + numbered sources. `LLMService.Generate`.
  5. Persist assistant Message (content with citations), mark completed; UsageLog: latency_ms, status, provider_id, error fields on failure.
  6. Return the answer + search results to the handler response.
- **Runtime metrics** (exposed to stats): current queue depth, in-flight count, total rejected count — kept in memory on the `AIHandler`, read by StatsService.
- Response shape for `/api/v1/ask` and `/api/v1/ask/url`: `{ data: { sessionId, messageId, answer, sources: [{position,title,url,domain,snippet}] } }`.

### 4.8 Handlers + router (full route table)

Register all routes in `internal/router/router.go` (Gin). Envelope on everything.

| Method | Path | Access | Handler behavior |
| --- | --- | --- | --- |
| GET | `/health` | public | `{ data: { status: "ok" } }` |
| POST | `/api/v1/ask` | public + rate limit | AI job (§4.7); daily quota + queue |
| POST | `/api/v1/ask/url` | public + **stricter** rate limit | validate URL (SSRF) → enqueue crawl → answer about that page |
| GET | `/api/v1/sessions/:id` | user | session + messages + `search_results` (deep links); `SESS01002` if not owner |
| GET | `/api/v1/me` | user | profile + today's usage + remaining quota |
| PATCH | `/api/v1/me` | user | update name/email |
| POST | `/api/v1/me/avatar` | user | multipart upload → avatar_url |
| GET | `/api/v1/site` | public | site_name, logo_url, tagline |
| POST | `/api/v1/auth/login` | public | `{ data: { token } }` |
| POST | `/api/v1/auth/logout` | owner | 200 envelope (token discarded client-side) |
| GET | `/api/v1/admin/users` | owner | search (`?q=`), paginated (`?page=&page_size=`) |
| POST | `/api/v1/admin/users` | owner | create user |
| PATCH | `/api/v1/admin/users/:id` | owner | update role/status/quota |
| DELETE | `/api/v1/admin/users/:id` | owner | delete user |
| GET | `/api/v1/admin/stats` | owner | user statistics |
| GET | `/api/v1/admin/stats/ai` | owner | AI statistics (§4.6) |
| GET | `/api/v1/admin/ai/providers` | owner | list providers |
| POST | `/api/v1/admin/ai/providers` | owner | create provider (api_key encrypted at rest) |
| PATCH | `/api/v1/admin/ai/providers/:id` | owner | update provider / set active |
| DELETE | `/api/v1/admin/ai/providers/:id` | owner | delete provider |
| GET | `/api/v1/admin/ai/queue-config` | owner | read `ai_queue_config` |
| PATCH | `/api/v1/admin/ai/queue-config` | owner | update (live, no restart) |
| GET | `/api/v1/admin/site-settings` | owner | get branding |
| PATCH | `/api/v1/admin/site-settings` | owner | update site_name/tagline |
| POST | `/api/v1/admin/site-settings/logo` | owner | upload/replace logo |

Validation: Gin binding + explicit checks in services. Every failure → Logrus log (internal cause) + sanitized envelope.

### 4.9 Security (`internal/services` / `internal/middleware`)

- **Rate limiting (Redis sliding window):** middleware `RateLimit(scope, max, window)`. Keys `ratelimit:{scope}:{key}` (key = user id or IP). Ask: e.g., 10/min. URL submission: stricter, e.g., 5/hour. Over limit → `AISY02002` (or a dedicated code) with friendly message.
- **SSRF guard** (in `CrawlService` before calling the crawler): parse URL; scheme must be `http`/`https`; resolve host; reject loopback/private/link-local ranges (127.0.0.0/8, 10/8, 172.16/12, 192.168/16, ::1, etc.) and non-standard ports. Rejected → mark `CrawlJob` `blocked` + `AISY03002`.
- **Secrets at rest:** AES-GCM using `ENCRYPTION_KEY` for provider `api_key_encrypted` (store base64 nonce+ciphertext). Round-trip unit test required.
- **QR payment codes** (security-scoped only): `POST /api/v1/qr/generate` can be a stub; signature = HMAC-SHA256(key, payload+expires_at); validate on read: not expired + signature matches → else reject.
- **Log hygiene:** sanitize query before persisting (`UsageLog.query`); never log email bodies, keys, or full queries.
- **Input sanitization:** trim/limit lengths on all user inputs.

---

## 5. Crawler sidecar (Node + Playwright)

`crawler/` is a tiny Node service (Express or plain `node:http`), only reachable inside the Docker network:

- `POST /fetch` body `{ url }` → runs Playwright (chromium, headless), waits for load, returns `{ title, text }` (text = `document.body.innerText` truncated to ~20k chars, stripped of nav/footer by heuristic if easy).
- Timeout per fetch (e.g., 20s). Errors → non-2xx JSON `{ error }`.
- Dockerfile: `mcr.microsoft.com/playwright:v1.x` base, `npm ci --omit=dev` (playwright is a runtime dep here), expose 3000.
- No auth needed internally (Go validates SSRF before calling).

---

## 6. Frontend (Vue 3 + TypeScript + Vite)

Scaffold with `npm create vite@latest frontend -- --template vue-ts`. Add `vue-router@4`, `pinia`. Dev proxy in `vite.config.ts`: `/api` → `http://localhost:8080` (so `VITE_API_BASE_URL=/api/v1` works locally too).

### 6.1 Design tokens & theming

- `src/styles/tokens.css`: `:root` (light) + `[data-theme="dark"]` custom properties — `--color-bg`, `--color-surface`, `--color-text`, `--color-text-muted`, `--color-border`, `--color-primary`, `--color-primary-contrast`, `--color-danger`, `--color-success`, `--color-warning`, `--radius-md`, `--space-*`, `--font-*`. **No hardcoded colors anywhere else.**
- `src/stores/theme.ts`: state `light | dark | system` (localStorage key `theme`); effective theme = system via `matchMedia('(prefers-color-scheme: dark)')` when `system`.
- **No flash of wrong theme:** in `index.html`, inline script BEFORE the app bundle sets `document.documentElement.dataset.theme` from localStorage/matchMedia synchronously. `main.ts` applies on changes.
- Theme toggle: `ThemeToggle` component (Light/Dark/System cycle or dropdown), placed in main-page header, account settings, and control panel header.

### 6.2 Reusable component catalog (`src/components/`)

Build once, use everywhere:

- `BaseButton.vue` (variants: primary/secondary/danger/ghost; sizes), `BaseInput.vue`, `BaseTextarea.vue`, `BaseSelect.vue`
- `BaseModal.vue` — backdrop, focus trap, Escape-to-close, title slot, body slot, footer slot; `EditModal.vue` (form wrapper: title, fields via slots, Save/Cancel, loading + error states) and `DeleteConfirmModal.vue` (entity name prop, warning copy, Delete/Cancel; emits `confirm` only on explicit click)
- `Tabs.vue` (tab list + content slot; horizontal scroll on mobile)
- `SidebarMenu.vue` — parent → child accordion (desktop) + drawer with backdrop (mobile via `isDrawer` prop); `expanded` state per parent; highlights active child; deep-link aware (router link active state)
- `ThemeToggle.vue`, `StatusPill.vue`, `Badge.vue`, `Avatar.vue` (image w/ fallback initials), `SkeletonLoader.vue`, `ErrorBanner.vue`, `LoadingIndicator.vue`, `EmptyState.vue`, `UsageBar.vue`, `Pagination.vue`

### 6.3 Views & routes (`src/router/index.ts`)

| Route | View | Notes |
| --- | --- | --- |
| `/` | `MainPage` | ask box + "Ask Me" → router.push(`/search?q=…`) |
| `/search` | `ResultPage` | reads `q`; renders summary + sources + follow-ups |
| `/account` | `AccountSettings` | tabs: Profile / AI Limit & Usage / Session |
| `/admin/login` | `AdminLogin` | posts `/auth/login`, stores token |
| `/admin` | `ControlPanel` | guard: no token → redirect `/admin/login`; layout: sidebar + content router-view |

Control panel child routes (deep links): `/admin/users`, `/admin/users/suspended`, `/admin/stats/overview`, `/admin/stats/top-queries`, `/admin/stats/per-user`, `/admin/stats/ai`, `/admin/ai/providers`, `/admin/ai/queue`, `/admin/branding/identity`, `/admin/branding/logo`. Parent auto-expands for the active child.

Views per PRD mockups (§4.1–§4.6 of the PRD):

- **MainPage**: centered `AskBox`, header (site name/logo from `/api/v1/site`), top-right `Avatar` button → `/account`, `ThemeToggle`.
- **ResultPage**: header with compact ask box; query header ("About N sources · X.Xs"); `AISummaryCard` (answer text with `[n]` citation links); numbered `SourceCard` list (favicon→domain initial, title, snippet, URL, `target="_blank"`); `SuggestedFollowUps` chips (submits as follow-up); follow-up thread (each follow-up = user bubble + summary + its own sources); skeleton loading while pending; error banner with retry. Follow-ups POST `/api/v1/ask` with `sessionId` in body and append inline.
- **AccountSettings**: `Tabs` with 3 tabs; Profile tab (avatar upload w/ preview + default fallback, name/email form); AI Limit & Usage tab (`UsageBar` used/total + friendly limit message); Session tab (account info + logout button → clear token → `/`).
- **ControlPanel**: `SidebarMenu` (desktop) / drawer (mobile) with parents Users / Statistics / AI Settings / Branding; header shows current page + `ThemeToggle` + logout. Each management page is a list w/ search/pagination + `[+ Add]` + per-row `[Edit]`/`[Delete]` opening the shared modals; destructive actions (delete, suspend) always confirm via `DeleteConfirmModal`. Statistics pages are read-only. AI Stats page: KPI cards (success rate, failures, avg latency, queue 3/20 · in-flight 4/8), filterable error table (code/type/count/last seen), latency p50/p95/p99 chart (simple CSS/SVG bars are fine — no chart lib required, or use one if already present).

### 6.4 API service & stores

- `services/api.ts`: fetch wrapper — base `VITE_API_BASE_URL`; attaches `Authorization` from auth store; parses envelope: `errorCode` non-empty → throw typed `ApiError { code, message }`; map known codes to friendly copy (e.g., `AISY02001` → "We're busy — try again in a moment.", `AISY02003` → "You've reached today's limit — try again tomorrow."). Typed methods per endpoint (§4.8).
- Stores (Pinia): `auth` (token, user, login/logout, expired-session handling — on 401 clear + redirect), `site` (public branding), `settings` (account), `admin` (users list/search/pagination, stats, ai stats, providers, queue config, site settings), `theme` (§6.1).

### 6.5 Responsive & dark-mode checklist (verify, don't skip)

- No horizontal scroll at 360px, 768px, and desktop; breakpoints: `<768` mobile, `768–1023` tablet, `≥1024` desktop (control panel sidebar persistent ≥1024, drawer below).
- Touch targets ≥44×44px; inputs ≥16px font (no iOS zoom).
- Tabs scroll horizontally on mobile; tables reflow to stacked cards; modals become near-full-width dialogs/bottom sheets.
- Result page ask box reachable while keyboard open; sticky bottom ask bar on small screens while scrolling.
- Dark mode passes on every page; WCAG AA contrast for text/UI in both themes; logo/avatar legible in both.

---

## 7. Docker & deployment

- **docker-compose.yml** services: `postgres`, `redis`, `searxng` (`searxng/searxng`, volume for `settings.yml`; enable `search.formats: [html, json]`), `crawler`, `api` (backend Dockerfile), `frontend` (nginx serving dist + proxying `/api` → api:8080), optional `ollama` (commented, for local LLM).
- All services on one internal network; only `frontend` (and optionally `api`) expose ports to the host. `searxng`, `crawler`, `postgres`, `redis` are **not** exposed publicly.
- **backend/Dockerfile**: multi-stage `golang:1.22` build → slim runtime; non-root user.
- **frontend/Dockerfile**: `node` build (with `VITE_API_BASE_URL=/api/v1`) → `nginx:alpine`; nginx config proxies `/api/` to `api:8080`.
- **docker-compose.prod.yml**: same stack, `restart: unless-stopped`, prod env from root `.env`, no dev volumes.
- `deploy-prod.sh` gates on `/health` (§3).

---

## 8. Tests (required minimum)

Backend — `go test ./...` must pass:
- Envelope helper; error-code format regex (`^[A-Z]{4}[0-9]{2}[0-9]{3}$`) over the whole `contracts` catalog.
- Config defaults; seed idempotency.
- Queue overflow → `AISY02001`; rate-limiter window logic; daily quota → `AISY02003`.
- SSRF guard rejects internal hosts (127.0.0.1, 10.x, localhost, ::1); accepts public URLs.
- SearXNG JSON → source-card mapping; graceful fallback when SearXNG is down (`AISY03001`).
- AES-GCM encryption round-trip; QR signature tamper/expiry rejection.
- RBAC: general user gets 403 on admin routes; AI stats aggregation (success rate, error grouping, p50/p95/p99).

Frontend — `npm run typecheck` + Vitest:
- `AskBox` submit emits question; `Tabs` switches content; `ResultPage` renders summary + source cards; `DeleteConfirmModal` only emits `confirm` on explicit click; theme store persists + applies before paint; API service maps `AISY02001` → friendly message.

Integration (at least via `go test` with httptest or manual `run-local.sh`):
- `/health` envelope; `/api/v1/site` returns seeded generic branding; ask → cited answer + sources; owner updates `ai_queue_config` → new values effective without restart.

---

## 9. Implementation order (do it in this sequence)

1. **Foundation**: `.env.example`, `.gitignore`, `run-local.sh`, `deploy-prod.sh`, `docker-compose.yml` (+prod), backend skeleton (config, contracts, middleware, models, AutoMigrate + seeds, router with stubs), frontend skeleton (Vite + router + pinia + tokens + theme bootstrap), crawler skeleton. Gate: `docker compose up` boots; `/health` OK; `go build ./...` + `go vet ./...` + typecheck pass; `/api/v1/site` returns seeded branding; main page renders generic site name.
2. **Main + Result page**: MainPage, ResultPage, account settings tabs, reusable components (`AskBox`, `SourceCard`, `Tabs`, `Avatar`, `UsageBar`…), API service, stores, responsive + dark mode. Gate: ask → Result Page renders (stubbed until step 3); both themes; ≥360px.
3. **AI pipeline**: AI handler + worker pool/queue, LLM service (Ollama + OpenAI-compatible), SearchService (SearXNG), CrawlService + crawler sidecar + SSRF guard, `search_results` persistence, `GET /sessions/:id`, rate limits + quota, `usage_logs` with error fields. Gate: question with web data → cited answer + sources; queue full → friendly busy; SearXNG down → graceful.
4. **Control panel**: auth + RBAC, sidebar parent→child + drawer, `EditModal`/`DeleteConfirmModal` + CRUD (users, providers), stats + **AI stats**, queue config live edits, branding, theme toggle. Gate: owner does full user CRUD with confirmations; sidebar deep links work on desktop + mobile widths; AI stats show real data.
5. **Hardening**: CORS validated, input sanitization, log hygiene, QR signing stub, `deploy-prod.sh` health gate verified on staging, WCAG AA + 360px pass, full `go test ./...` + frontend suite green.

---

## 10. Definition of Done / final verification

- `docker compose up` boots the full stack locally; `GET /health` → `{ data: { status: "ok" } }`.
- Ask a question → Result Page shows cited summary + numbered sources (persisted in `search_results`); follow-ups + suggested chips work; URL submission rate-limited; queue full shows friendly message.
- Account settings tabs work; avatar uploads; usage bar reflects quota; logout works.
- Owner logs in → sidebar parent→child (accordion/drawer) with deep links; full CRUD via reusable modals with delete confirmations; AI stats page shows success rate, error list, latency percentiles, queue health; AI/branding changes apply live.
- Dark mode + ≥360px responsive verified on all pages; WCAG AA contrast in both themes.
- `go test ./...`, `go vet ./...`, frontend typecheck + tests green.
- No `.vue` file > 1000 lines; no duplicated component markup; no hardcoded branding or colors; no secrets committed.

## 11. After implementation

Per `AGENTS.md`, update as needed: API contract + error catalog under `docs/tech_stack/` (create it), `README.md` (keep it as the onboarding entry point), and this file if any decision changed. Do not delete the PRD/SDD/Sprint Plan.
