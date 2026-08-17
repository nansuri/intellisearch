# Software Design Document — AI-Powered Jastip Platform

| Field | Value |
| --- | --- |
| **Product Name** | Intellisearch (generic — branding configurable by the owner) |
| **Document Version** | 1.5 (generic open source; no hardcoded branding) |
| **Status** | Target architecture; implementation status tracked separately |
| **Last Updated** | 2026-08-17 |
| **Related Documents** | [PRD index](../PRD/index.md) · [Implementation status](../tech_stack/IMPLEMENTATION_STATUS.md) |

> The SDD is split by concern. This index holds the shared technical context (stack, layering, milestones). Detailed design lives in the files linked below.

**Document History**

| Version | Date | Changes |
| --- | --- | --- |
| 1.1 | 2026-08-17 | Initial generic open-source release |
| 1.2 | 2026-08-17 | Added SearXNG self-hosted metasearch for web search (backend-to-backend); crawler retained for page content and URL submissions |
| 1.3 | 2026-08-17 | Mirrored PRD UI requirements: Result Page, tabbed Account Settings, sidebar parent → child control-panel menu, CRUD + reusable modals, responsive/mobile-first, dark mode via design tokens; added `search_results` entity and `GET /api/v1/sessions/:id` |
| 1.4 | 2026-08-17 | Added detailed AI statistics: `usage_logs` extended with typed error fields, AI runtime queue metrics, `GET /api/v1/admin/stats/ai` |
| 1.5 | 2026-08-17 | Recorded implementation status, local SQLite / production PostgreSQL split, persisted foundation entities, JWT account APIs, SearXNG client, SSRF guard; split into per-concern docs |

---

## 1. Introduction

This document series is the technical companion to the PRD: architecture, data models, API, and implementation details for the single-shot MVP. Product behavior and UI are defined in the PRD and not repeated here.

The platform is a generic, open-source project. Branding (site name, logo, tagline) is stored in `site_settings` and must never be hardcoded; the codebase ships with a generic default that is overridden per deployment.

## 2. Chosen Tech Stack

| Layer | Technology | Notes |
| --- | --- | --- |
| **Frontend** | Vue.js (DDD Pattern) | Component-based with domain-driven folder structure |
| **Backend** | Go + Gin Gonic + GORM + Logrus | RESTful API with DDD layering (Handler → Service → Repository) |
| **Database** | SQLite locally; PostgreSQL in Docker/production + Redis | SQLite keeps host-run development self-contained; production uses PostgreSQL; Redis remains required for caching and rate-limiting |
| **Search** | SearXNG (self-hosted metasearch) | Container on the same server; JSON API called backend-to-backend; no third-party search API keys |
| **AI Integrations** | Puppeteer/Playwright for scraping; Ollama (local) for LLM chat | Privacy-first, no cloud dependency |
| **Deployment** | Docker + docker-compose | Local server deployment — `run-local.sh` and `deploy-prod.sh` |

Additional considerations:

- **Auth:** JWT-based sessions for the control panel.
- **Secrets:** environment variables / `.env` files, never committed to the repo.

## 3. Architecture Pattern (both tiers use DDD)

- **Frontend (Vue.js):** `views/` → `domains/{feature}/components/` → `domains/{feature}/composables/` → `stores/`
- **Backend (Go):** `handlers/` → `services/` → `repositories/` → `models/entities/`
  - One AI-specialized handler; all AI requests go through this interface.
  - The AI handler owns the concurrency config and queue pool.

### Rules

- Never write a `.vue` file longer than 1000 lines; keep files well below it.
- For the FE, always create reusable components (button, modal, badge, table, etc.) instead of duplicating markup.

## 4. Document Index

| Concern | File |
| --- | --- |
| System architecture & request flow (incl. SearXNG + crawler) | [architecture.md](architecture.md) |
| Frontend UI architecture | [frontend-architecture.md](frontend-architecture.md) |
| AI concurrency & queue, pipeline | [ai-pipeline.md](ai-pipeline.md) |
| Data models & taxonomies | [data-models.md](data-models.md) |
| API design overview | [api-design.md](api-design.md) |
| Security implementation | [security.md](security.md) |
| Non-functional requirements & milestones | [nfr-milestones.md](nfr-milestones.md) |