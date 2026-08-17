# SDD — AI Concurrency, Queue & Pipeline

> Part of the [SDD index](index.md).

## 1. Single AI Interface

A single **AI handler** is the only entry point for all AI work. All AI requests go through this one handler, backed by a worker pool + queue (config from `ai_queue_config`).

## 2. Concurrency & Queue Design

- A **worker pool** processes jobs with at most `max_concurrent` in flight.
- Incoming jobs beyond the workers wait in a **queue** of at most `max_queue_size`; overflow is rejected with a friendly "busy, try again" message.
- Job timeout enforced by `request_timeout_ms`; timed-out jobs are marked `failed` in `usage_logs`.
- All knobs live in `ai_queue_config` (singleton row) and are editable from the Owner Control Panel **without redeploying**.
- The AI handler exposes **runtime metrics** for the statistics panel: current queue depth, in-flight count vs. `max_concurrent`, and cumulative rejected/overflowed requests (kept in-process, exposed via the admin AI-stats endpoint).

## 3. Pipeline Stages

1. **Validate & rate-limit** the request (Redis sliding window).
2. **Enqueue** the job; the worker pool picks it up within `max_concurrent` slots.
3. **Search** the web for sources:
   - Search listings via the **Search service** → self-hosted SearXNG container (`GET http://searxng:8080/search?q=…&format=json`, backend-to-backend over the Docker network).
   - For each needed source, the **Crawler service** (Puppeteer/Playwright) fetches the page content for full-text grounding.
   - URL submissions skip search and use the crawler directly.
4. **Synthesize** — the LLM (Ollama local / OpenAI-compatible provider) produces a cited answer from the query + crawled content.
5. **Persist & return** — write session/message/`search_results`, kill the crawl job, return the envelope to the UI.

## 4. Ollama / LLM Requirements

- Local Ollama (privacy-first; no user data leaves the server).
- Model and parameters are configured via the AI providers table (provider_type `ollama` or `openai_compatible`; `base_url`, `model`, `parameters`), and the active provider is toggled in the control panel without redeploying.