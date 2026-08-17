# SDD — System Architecture & Request Flow

> Part of the [SDD index](index.md). Product behavior lives in the [PRD](../PRD/index.md).

## 1. High-Level Flow

```
Browser (Vue.js)
   │  POST /api/v1/ask
   ▼
Go API (Gin) ── handlers/ ── services/ ── repositories/ ── PostgreSQL
   │
   ├── AI Handler (single interface for all AI requests)
   │      └── Worker Pool + Queue (config from ai_queue_config)
   │             ├── LLM Service ──► Ollama (local) / OpenAI-compatible
   │             ├── Search Service ──► SearXNG container (JSON API) ──► upstream engines
   │             └── Crawler Service ──► Puppeteer/Playwright (page content)
   │
   └── Redis ── rate limiting, queue state
```

- The API uses **SQLite** for host-run development (`DB_DRIVER=sqlite`, keep everything local and self-contained) and **PostgreSQL** for Docker/production (`DB_DRIVER=postgres`). The GORM domain models and migrations are shared between both drivers.

## 2. Ask Flow (end to end)

1. User submits a question from the main page.
2. The Go API validates & rate-limits the request (Redis).
3. The AI handler enqueues the job; the worker pool executes it with `max_concurrent` slots.
4. If web data is required, the **Search service** queries SearXNG for source listings; the **Crawler service** fetches the top pages' content; the LLM (Ollama) then synthesizes a cited answer.
5. The answer is persisted (session, message, `search_results`) and returned to the UI.

> A plain-language version of this flow with a sequence diagram lives in [PRD — Main Page & Ask Flow](../PRD/main-page.md#what-happens-in-the-backend-when-a-user-searches).

## 3. Web Search & Page Content (SearXNG + Crawler)

- **Search listings** come from a self-hosted **SearXNG** container (`searxng/searxng`) reached backend-to-backend over the internal Docker network: `GET http://searxng:8080/search?q=…&format=json`. JSON output is enabled in SearXNG's `settings.yml` (`search.formats: [html, json]`).
- SearXNG aggregates upstream engines (Google, Bing, DuckDuckGo, …) and returns structured results (`title`, `url`, `content` snippet, `engine`, `score`) — these become the Result Page's numbered source cards (PRD §3.4).
- **Page content** for deep reading is fetched by the **Crawler service** (Puppeteer/Playwright): the AI handler gets listings from SearXNG, then fetches the top N source pages so the LLM can write a cited summary grounded in full text rather than snippets alone.
- **Explicit URL submissions** (`POST /api/v1/ask/url`) use the Crawler service only; the search flow is not involved.
- **Resilience:** if SearXNG is unreachable, the request degrades gracefully — the LLM answers without web sources, or the standard error banner with retry is shown.
- **Config:** `SEARXNG_BASE_URL` (e.g., `http://searxng:8080`) lives in the shared root `.env`; engines/categories/language/safe-search are tuned in SearXNG's `settings.yml`; Redis rate limits on the ask endpoints apply on top of SearXNG's own limiter.
- **Security:** SearXNG runs as an internal container reachable only inside the Docker network (never publicly exposed). Calls to it are trusted internal traffic; the SSRF guard still protects the crawler for URL submissions.