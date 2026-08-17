# API Contracts

## Response envelope

Every API response uses:

```json
{ "data": {}, "errorCode": "", "errorMessage": "" }
```

## Implemented endpoints

| Method | Path | Access | Description |
| --- | --- | --- | --- |
| GET | `/health` | public | Service health envelope. |
| GET | `/api/v1/site` | public | Reads the public site identity from the seeded `site_settings` database row. |
| POST | `/api/v1/auth/login` | public | Validates an active account and returns a signed JWT plus the safe user profile. Serves both the general-user sign-in page (`/login`) and the Super Owner control-panel sign-in page (`/admin/login`); role checks happen in the frontend flow and on every admin route. |
| POST | `/api/v1/auth/logout` | Super Owner | Confirms client-side session disposal. |
| GET | `/api/v1/me` | user | Returns the current user's safe profile. |
| PATCH | `/api/v1/me` | user | Updates the current user's name and email. |
| POST | `/api/v1/ask` | public (rate-limited) | Submits a question to the AI pipeline (worker pool + bounded queue). Body: `{ query, sessionId?, location? }` where optional `location` is `{ latitude, longitude, accuracy? }` from the browser Geolocation API. When the query mentions nearby intent (e.g. "near me", "near my place"), the backend reverse-geocodes the coordinates and enriches the SearXNG search query. Coordinates are not persisted in messages or usage logs. Response: `{ data: { sessionId, messageId, answer, sources } }`. Follow-ups pass the `sessionId` to continue the same conversation. |
| POST | `/api/v1/ask/url` | public (stricter rate limit, 5/hour) | Crawls a submitted URL and answers about that page. Body: `{ url }`. Response shape matches `/ask`. URLs are SSRF-guarded (scheme + host validation, internal/private ranges blocked). |
| GET | `/api/v1/sessions/:id` | public (anonymous sessions) or owner (signed-in sessions) | Returns one chat session with message history and per-message `sources`. Anonymous sessions (`userId` null) are readable by id without auth; owned sessions require the owning user. |
| POST | `/api/v1/me/avatar` | user | Uploads the user's avatar image (JPG/PNG/GIF/WebP ≤ 2 MB) and returns `{ avatarUrl }`. |
| GET | `/api/v1/admin/users` | Super Owner | Lists/search users. Query: `q`, `page`, `page_size`. Response: `{ users, total, page, pageSize }`. |
| POST | `/api/v1/admin/users` | Super Owner | Creates a user. Body: `{ name, email, password, role, aiDailyQuota }`. |
| PATCH | `/api/v1/admin/users/:id` | Super Owner | Updates role/status/`aiDailyQuota` (partial body allowed). |
| DELETE | `/api/v1/admin/users/:id` | Super Owner | Deletes a user with history. |
| GET | `/api/v1/admin/stats` | Super Owner | Usage overview: `{ questionsToday, questionsWeek, activeUsersWeek, failed, topQueries[], perUserUsage[] }`. |
| GET | `/api/v1/admin/stats/trends` | Super Owner | Daily/weekly question counts for charts: `{ daily[7] {label,count}, weekly[8] {label,count} }`. Buckets computed in the service so behavior is identical on SQLite and PostgreSQL. |
| GET | `/api/v1/admin/stats/ai` | Super Owner | AI stats: `{ totalCompleted, totalFailed, successRate, errors[], latency{...}, providers[], queue{...} }`. Optional `type` filters the error list. |
| GET | `/api/v1/admin/ai/providers` | Super Owner | Lists AI providers. |
| POST | `/api/v1/admin/ai/providers` | Super Owner | Creates a provider (type, base URL, model, parameters, optional `apiKey`, `isActive`). |
| PATCH | `/api/v1/admin/ai/providers/:id` | Super Owner | Updates a provider (blank `apiKey` keeps the existing encrypted key). |
| DELETE | `/api/v1/admin/ai/providers/:id` | Super Owner | Deletes a provider. |
| GET | `/api/v1/admin/ai/queue-config` | Super Owner | Reads `ai_queue_configs` singleton. |
| PATCH | `/api/v1/admin/ai/queue-config` | Super Owner | Updates `max_concurrent`, `max_queue_size`, `request_timeout_ms`, `per_user_rate_limit` (applies within ~5 s, no redeploy). |
| GET | `/api/v1/admin/site-settings` | Super Owner | Reads branding: `{ siteName, logoUrl, faviconUrl, tagline }`. |
| PATCH | `/api/v1/admin/site-settings` | Super Owner | Updates `siteName` and `tagline`. |
| POST | `/api/v1/admin/site-settings/logo` | Super Owner | Uploads the site logo (returns `{ logoUrl }`). Publically cached for a short TTL. |
| DELETE | `/api/v1/admin/site-settings/logo` | Super Owner | Removes the site logo so the public pages fall back to the default initials mark (returns `{ logoUrl: null }`). |
| POST | `/api/v1/admin/site-settings/favicon` | Super Owner | Multipart upload of the site favicon (JPG/PNG/GIF/WebP, ≤ 2 MB; returns `{ faviconUrl }`). The browser tab icon updates immediately. |
| DELETE | `/api/v1/admin/site-settings/favicon` | Super Owner | Removes the favicon so browsers fall back to the bundled default SVG (returns `{ faviconUrl: null }`). |

Error-code constants are defined in `backend/internal/contracts/errors.go` and follow `<FEATURE><SUBSET><ERROR>` (e.g. `AISY02001` queue full, `AISY02002` rate limited, `AISY02003` daily quota exceeded, `AISY03002` URL blocked, `AISY03003` invalid URL, `AISY03004` crawl failed, `AISY01001/2/3` provider unavailable/timeout/error). Admin panel errors use `ADMN01001` (user ops), `ADMN02001` (statistics), `ADMN03001/2` (providers), `ADMN04001` (queue), `ADMN05001/2` (site settings/logo), `ADMN05003` (favicon upload).

### AI pipeline behavior

- **Queue & concurrency:** knobs (`max_concurrent`, `max_queue_size`, `request_timeout_ms`, `per_user_rate_limit`) are read from the `ai_queue_configs` singleton with a 5-second cache, so Owner Control Panel edits apply without restarting. Queue overflow returns `AISY02001` with a friendly "busy, try again" message.
- **Rate limiting (Redis sliding window):** applies per user (or per IP for anonymous callers) via `ratelimit:ask:{key}`. URL submission is limited to 5/hour regardless of the configured per-user limit.
- **Daily quota:** authenticated users with a personal `ai_daily_quota` are limited to that many asks per UTC day (`AISY02003`); `0` means unlimited.
- **Graceful degradation:** if SearXNG is unreachable the LLM still answers without web sources. Every ask persists a `chat_sessions` row, user/assistant `messages`, `search_results` (source cards), and a `usage_logs` row with status/error/latency for the statistics panel.

## Crawler safety

Both the API service layer and crawler sidecar reject non-HTTP(S), loopback, private-network, link-local, and local/internal host targets. URL submissions are recorded as `crawl_jobs` (`blocked` when the SSRF guard rejects them).

## Database environments

Host-run local development uses SQLite (`DB_DRIVER=sqlite`, `DB_SQLITE_PATH=./backend/data/intellisearch.db`). Production and Docker API services use PostgreSQL (`DB_DRIVER=postgres`). GORM migrations and seeds are shared between both drivers. If Redis is unreachable at startup the API logs loudly and runs with rate limiting disabled (`NoopLimiter`) so the service keeps serving.
