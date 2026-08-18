# Sprint Plan: Intellisearch (Single-shot MVP)

| Field | Value |
| --- | --- |
| **Product** | Intellisearch (generic, white-label) |
| **Version** | 1.2 |
| **Status** | Active |
| **Related** | [PRD](../../PRD/index.md) · [SDD](../../SDD/index.md) |

**Change Log**

| Version | Date | Changes |
| --- | --- | --- |
| 1.0 | 2026-08-17 | Initial sprint plan |
| 1.1 | 2026-08-17 | Sprints 1, 2, 4, 5 and Definition of Done extended: Result Page, tabbed Account Settings, parent → child sidebar with CRUD + reusable modals, responsive + dark mode (design tokens) |
| 1.2 | 2026-08-17 | Sprint 4 extended with detailed AI statistics (success rate, error list, latency percentiles, queue health) via `usage_logs` error fields + AI runtime metrics |

The MVP is delivered in five sprints aligned with the SDD milestones (M1–M5). Each sprint has a scope, a task list, test coverage, and exit criteria. Hard rules apply throughout: `.vue` files ≤ 1000 lines, reusable components, `{ data, errorCode, errorMessage }` envelope, typed error codes in `backend/internal/contracts`, Logrus logging, no hardcoded branding, and no hardcoded colors (all UI colors from design tokens for light/dark themes).

Current delivery is incremental; see [`docs/tech_stack/IMPLEMENTATION_STATUS.md`](../IMPLEMENTATION_STATUS.md) for the code-backed status. Unchecked tasks are not yet available in the running application.

---

## Sprint 1 — Scaffolding (SDD M1)

**Goal:** Repo skeleton, Docker stack, and local scripts so `docker compose up` boots API + PostgreSQL + Redis locally.

**Tasks**
- [x] Fix planning docs and add README entry point
- [~] Backend foundation: config from root `.env`, PostgreSQL connection, GORM AutoMigrate + idempotent seeds for the current foundation entities, Logrus setup
- [~] Backend DDD layers present: `contracts/` (error codes), `handlers/`, `services/`, `repositories/`, `models/` (foundation + auth/account paths implemented)
- [ ] Response envelope helpers + CORS, request logging, recovery middleware
- [x] Seed singleton rows (`ai_queue_config`, `site_settings` with generic branding)
- [ ] Route table per SDD §6 registered (stubs return typed "not implemented" errors)
- [ ] Frontend skeleton: Vue 3 + TS + Vite + Router + Pinia, modular styles, reusable `BaseButton`/`BaseInput`
- [ ] Design tokens (light + dark) and theme bootstrap (system default, applied before first paint); reusable `BaseModal`, `Tabs`, `ThemeToggle` primitives
- [ ] `docker-compose.yml`, `docker-compose.prod.yml`, `backend/Dockerfile`, `frontend/Dockerfile` (nginx), `run-local.sh`, `deploy-prod.sh`, `.env.example`, `.gitignore`

**Tests**
- Unit: envelope helper, error-code format (`<FEATURE><SUBSET><ERROR>`), config defaults
- Integration: health endpoint returns envelope `{ data: { status: "ok" } }`; `/api/v1/site` returns seeded generic branding

**Exit criteria**
- `docker-compose up` boots API + DB + Redis locally; `GET /health` returns 200
- `go build ./...` and `go vet ./...` pass; frontend typecheck passes
- Public main page renders the generic site name from `site_settings`

---

## Sprint 2 — Main Page & Result Page UI (SDD M2)

**Goal:** Google-like main page, Result Page (cited summary + web sources + follow-ups), and tabbed account settings — responsive and in both themes — wired to the API.

**Tasks**
- [ ] Main page view: centered ask box, header with site name/logo from `/api/v1/site`, top-right avatar button
- [ ] Result Page view: query header, cited AI summary card, numbered web-sources list (SearXNG listings mapped to source cards), follow-up thread, suggested follow-up chips, persistent ask box (uses `/api/v1/ask`)
- [ ] Reusable components: `AskBox`, `ChatThread`, `MessageBubble`, `LoadingIndicator`, `ErrorBanner`, `SourceCard`, `SuggestedFollowUps`, `ThemeToggle`
- [ ] Account settings view: tabbed sub-menus — Profile (avatar upload w/ preview), AI Limit & Usage (usage bar), Session (logout) (uses `/api/v1/me`)
- [ ] Frontend typed API service with envelope parsing (`services/api.ts`)
- [ ] Responsive: mobile-first layouts (ask box, result page, settings) verified at ≥360px; touch targets ≥44×44px
- [ ] Dark mode: token-driven colors, Light/Dark/System toggle persisted, no flash of wrong theme
- [ ] Basic error handling: friendly "busy, try again" for queue-full errors

**Tests**
- Component tests: AskBox submit → emits question; ChatThread renders messages; Result Page renders summary + source cards; Account Settings tabs switch content; ErrorBanner on failure
- Integration: submit question → Result Page renders cited answer + sources; API error surfaces friendly message
- Responsive/theme: no horizontal scroll at 360px; theme toggle persists and applies before first paint

**Exit criteria**
- Ask a question → Result Page with cited answer + web sources renders in the browser (stubbed until Sprint 3 wiring)
- Account settings tabbed sub-menus read and display the current user's profile and usage
- All pages usable at ≥360px and in both light/dark themes

---

## Sprint 3 — AI Pipeline (SDD M3)

**Goal:** Single AI handler with worker pool + queue, Ollama integration, SearXNG web search, and web crawler.

**Tasks**
- [ ] AI handler (only entry point for AI work) with configurable worker pool + bounded queue from `ai_queue_config`
- [ ] LLM service: Ollama + OpenAI-compatible provider abstraction; job timeout → `failed` in `usage_logs`
- [ ] Search service: SearXNG container (JSON API) queried backend-to-backend; listings become the result page source cards
- [ ] Crawler service: Puppeteer/Playwright fetch + summarize (deep-reads top sources, handles URL submissions); SSRF guard (scheme/host validation)
- [ ] `POST /api/v1/ask` and `POST /api/v1/ask/url` with Redis rate limiting (stricter on URL submission)
- [~] Persist `chat_sessions`, `messages` (queued → streaming → completed/failed), `search_results` (source cards per assistant message), and `usage_logs` with typed error fields (models + repositories implemented; ask pipeline wiring remains)
- [ ] Per-user daily quota + friendly limit message
- [ ] `GET /api/v1/sessions/:id` returns a session with its messages + `search_results` (deep-linked result pages)

**Tests**
- Unit: queue overflow rejects with busy error; rate limiter window logic; SSRF guard rejects internal hosts; SearXNG JSON mapped to source cards; graceful fallback when SearXNG is down
- Integration: ask question with web data returns a combined cited answer with sources; quota exceeded returns limit error

**Exit criteria**
- Question with web data returns a combined, cited answer with a list of web sources
- Queue/concurrency knobs take effect without redeploying

---

## Sprint 4 — Owner Control Panel (SDD M4)

**Goal:** Auth + admin panel with a parent → child sidebar and full CRUD via reusable modals: user management, statistics, AI settings, branding.

**Tasks**
- [x] Auth: control-panel login/logout, JWT sessions, expired-session handling, role middleware (Super Owner only)
- [x] Sidebar navigation: parent → child accordion (desktop) + hamburger-triggered slide-in drawer with backdrop below 1024 px; deep-linkable child routes (e.g., `/admin/users/suspended`, `/admin/stats/visitors`); active child highlighted, active parent auto-expanded
- [x] Reusable modals: `BaseModal` + `ConfirmModal` (create/edit forms and destructive confirmation), shared by users and providers; reusable `FormField`, `PageHeader`, `StatusBadge`, `PaginationBar`, `StatCard`, `Avatar`, `EmptyState`, `LoadingSpinner`, `TrendChart`
- [x] User management: full CRUD — add/edit/delete users via modals, searchable + paginated list, suspend/reinstate with confirmation required
- [x] Statistics (read-only): questions per day/week, active users, top queries, per-user usage, daily/weekly trend charts
- [x] AI statistics: success/failure rate (overall + per provider/model), error grouping (typed error codes from `usage_logs`, filterable by type), latency percentiles (p50/p95/p99), queue health (depth, in-flight, rejected)
- [x] AI settings UI: provider CRUD (add/edit/delete, set active provider, model parameters, Ollama endpoint) via modals; concurrency/queue config (live)
- [x] Branding settings UI: site name, logo upload/drag-drop + removal (falls back to default), tagline (live, no redeploy)
- [x] Theme toggle in the panel header; panel renders in both themes
- [x] Secrets: provider API keys encrypted at rest (AES-GCM, key from env); never logged

**Tests**
- Unit: RBAC denies General Users on admin routes; encryption round-trip; delete requires confirmation; AI stats aggregates `usage_logs` (success rate, error grouping, latency percentiles)
- Integration: owner updates `ai_queue_config` → new values take effect without restart; sidebar accordion + deep links navigate correctly on desktop and mobile widths

**Exit criteria**
- Owner manages users with full CRUD via the reusable modals and changes AI + branding settings live
- Sidebar parent → child navigation works on desktop and mobile (drawer) with deep links

---

## Sprint 5 — Hardening & Deploy (SDD M5)

**Goal:** Security hardening and verified production deployment.

**Tasks**
- [ ] Rate limiting tuned; suspicious crawls marked `blocked`
- [ ] QR payment code generation: short expiry + HMAC signature, tamper/expiry rejection (security-scoped only)
- [ ] Input sanitization across endpoints; log hygiene (sanitized queries, no secrets/PII)
- [ ] CORS validated; API accepts only configured origins
- [ ] `deploy-prod.sh` verified: health check gates deployment success

**Tests**
- Security: tampered QR rejected; expired QR rejected; crawler cannot reach internal services; oversized/payload abuse rejected
- Theming/responsive: WCAG AA contrast verified in light and dark themes; all pages pass at ≥360px
- E2E: production stack deploys via `deploy-prod.sh` and `/health` reports ok

**Exit criteria**
- All SDD §11 technical acceptance criteria pass on staging
- MVP is deployable and the main page branding comes from `site_settings`

---

## Definition of Done (every sprint)

- All tasks complete with tests green (`go test ./...`, frontend test suite, typecheck)
- API contract / error catalog updated in `docs/tech_stack/` for any endpoint changes
- No `.vue` file exceeds 1000 lines; reusable components used instead of duplication
- No secrets in code; all env values documented in `.env.example`
- No hardcoded colors: all UI colors come from design tokens (light/dark)
