# SDD — Non-Functional Requirements & Milestones

> Part of the [SDD index](index.md).

## 1. Non-Functional Requirements

- **Performance:** direct answers fast; web-combined answers within a reasonable time; page load fast.
- **Reliability:** graceful degradation when the AI provider is down (clear error + retry, no crash).
- **Scalability:** concurrency/queue tuning without redeploy; horizontal scaling of the API behind the Redis-backed rate limiter.
- **Privacy:** privacy-first — Ollama runs locally; no user data sent to external LLM clouds.
- **Observability:** structured logs (Logrus) + metrics (usage_logs) for the statistics panel.
- **Responsiveness:** mobile-first; every page usable at ≥360px with no horizontal scrolling; touch targets ≥44×44px; inputs ≥16px (PRD §5).
- **Theming:** light and dark themes driven by design tokens; both meet WCAG AA contrast; theme follows the OS setting with a persisted manual override and no flash of wrong theme (PRD §5.5).

## 2. Milestones & Delivery (Single-shot MVP)

| Milestone | Scope | Exit Criteria |
| --- | --- | --- |
| M1 — Scaffolding | Repo, Docker, docker-compose, run-local.sh / deploy-prod.sh | `docker-compose up` boots API + DB + Redis locally |
| M2 — Main page & Result Page | Google-like UI, ask box, Result Page (cited summary, web sources, follow-ups), tabbed account settings, responsive + dark mode | Ask a question → cited answer + sources render in browser; responsive ≥360px and both themes verified |
| M3 — AI pipeline | AI handler, worker pool + queue, Ollama integration, SearXNG search integration, crawler | Question with web data returns a combined, cited answer with sources |
| M4 — Control panel | Auth, sidebar parent → child menu, CRUD + reusable modals, user management, statistics (incl. detailed AI stats), AI settings, branding | Owner manages users with full CRUD via reusable modals and changes AI/branding config live; sidebar works on desktop + mobile |
| M5 — Hardening & deploy | Rate limiting, secrets vaulting, QR signing, deploy-prod.sh verified | Technical acceptance criteria (§3) pass on staging |

## 3. Technical Acceptance Criteria

- API keys (Telegram/LLM) are stored only as encrypted/vaulted secrets.
- QR payment codes expire and reject tampered signatures.
- The crawler rejects URLs that reach internal services (SSRF guard).
- Search listings come from the self-hosted SearXNG container over the internal Docker network (never publicly exposed); the crawler handles page content and URL submissions.
- Asking a question renders the Result Page: a cited AI summary card and a numbered web-sources list (from SearXNG listings, persisted in `search_results`), with follow-up support and a persistent ask box.
- The app is fully usable at ≥360px (no horizontal scrolling) and renders correctly in light and dark mode with token-driven colors (no hardcoded colors; WCAG AA contrast).
- Account Settings uses tabbed sub-menus; the control panel sidebar supports parent → child menus (accordion on desktop, drawer on mobile) with deep-linkable child routes.
- The Owner Control Panel shows detailed AI statistics (success/failure rate, filterable error list, latency percentiles, queue health) computed from `usage_logs` and AI runtime metrics.
- Every management menu supports CRUD via the shared reusable Edit and Delete-confirmation modals; destructive actions (delete, suspend/block) always require confirmation.
- All files respect the 1000-line rule; FE reuses components.
- The public main page renders its branding from `site_settings`, not from hardcoded values; the shipped default is generic.