# SDD — Data Models & Taxonomies

> Part of the [SDD index](index.md).

## 1. Core Entities

### users
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| name | varchar | |
| email | varchar (unique) | |
| password_hash | varchar | nullable if OAuth later |
| role | enum (user_role) | default: general_user |
| status | enum (user_status) | active / suspended |
| created_at | timestamptz | |
| last_login_at | timestamptz | |
| avatar_url | varchar | nullable; uploaded image or default avatar |
| ai_daily_quota | int | daily question cap; 0 = unlimited. Newly registered accounts start at `ai_queue_config.default_daily_quota` (default 3); editable per user from the Owner Control Panel |

### chat_sessions
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| user_id | UUID (FK → users) | |
| title | varchar | = first question |
| created_at / updated_at | timestamptz | |

### messages
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| session_id | UUID (FK → chat_sessions) | |
| role | enum (message_role) | system / user / assistant |
| content | text | |
| status | enum (message_status) | queued / streaming / completed / failed |
| created_at | timestamptz | |

### search_results
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| message_id | UUID (FK → messages) | the assistant message that cited these sources |
| position | int | citation number ([1], [2], …) |
| title | varchar | from SearXNG listing |
| url | text | |
| domain | varchar | derived from URL |
| snippet | text | from SearXNG listing |
| created_at | timestamptz | |

> `search_results` persists the Result Page's source cards so shared/deep-linked result URLs and follow-up threads can re-render their sources without re-crawling.

### image_results
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| message_id | UUID (FK → messages) | the assistant message that searched for these images |
| position | int | grid order |
| title | varchar | from SearXNG image listing |
| url | text | the original page/asset URL |
| thumbnail_url | text | thumbnail (falls back to the full image URL) |
| source | varchar | originating site |
| width / height | int | from SearXNG `resolution` (0 when unknown) |
| created_at | timestamptz | |

> SearXNG **image results** (`categories=images`), persisted per assistant message so the result page can re-render the image grid on restore. Only URLs are stored — the API never fetches third-party images server-side (no SSRF surface; the browser loads them directly). Image search is best-effort in both ask modes and never fails an ask.

### map_points
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| message_id | UUID (FK → messages) | the assistant message that searched for these places |
| position | int | 0 = map center (the user's location), 1..N = nearby results |
| label | varchar | reverse-geocoded place label (center) or geocoded result name |
| latitude / longitude | double | coordinates |
| created_at | timestamptz | |

> **Map markers** for location-aware asks ("hospital near me"): the center is the user's shared position (reverse-geocoded label) and the markers are the top 5 source titles geocoded via Nominatim search, kept only when within 100 km of the user (max 6). Persisted per assistant message so the result page can re-render the interactive map (Leaflet + OSM tiles, no API key) on restore. Best-effort by design — a geocoding failure means fewer markers, never a failed ask; coordinates never appear in `usage_logs`.

### ai_providers
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| name | varchar | e.g., "local-ollama" |
| provider_type | enum (provider_type) | ollama / openai_compatible / pollinations / huggingface |
| base_url | varchar | e.g., http://ollama:11434 |
| model | varchar | |
| parameters | jsonb | temperature, max_tokens, context window |
| api_key_encrypted | text | nullable; encrypted at rest (see security.md) |
| is_active | boolean | only one active at a time |
| created_at / updated_at | timestamptz | |

### ai_queue_config (singleton row)
| Field | Type | Notes |
| --- | --- | --- |
| id | int (PK) | single row |
| max_concurrent | int | |
| max_queue_size | int | |
| request_timeout_ms | int | |
| per_user_rate_limit | int | questions per minute per user; 0 = unlimited |
| suggestion_cache_hours | int | hours the AI-composed history suggestions are reused per user before recomposing; 0 = always compose fresh. Editable from the Owner Control Panel (Queue & limits). |
| default_daily_quota | int | daily AI-usage quota granted to newly registered accounts (password or Google SSO); 0 = unlimited. Default 3. Editable from the Owner Control Panel (Queue & limits); affects only accounts created afterwards. |
| max_image_results | int | caps image results returned/persisted per web-search ask; 0 = unlimited. Default 20. Only the primary search of a thread fetches images (follow-ups skip). Editable from the Owner Control Panel (Queue & limits). |
| session_ttl_hours | int | how long a signed-in JWT session lasts before re-sign-in; 0 = fall back to the deployment `JWT_TTL_HOURS` (default 168). Default 168. Editable from the Owner Control Panel (Queue & limits → Account session); applies to sessions signed in after the change — already-issued tokens keep their original expiry |

### site_settings (singleton row)
| Field | Type | Notes |
| --- | --- | --- |
| id | int (PK) | single row |
| site_name | varchar | shown as logo/title on the main page; generic default shipped in code |
| logo_url | varchar | nullable; uploaded logo or default |
| tagline | varchar | nullable; optional short description |
| copyright | varchar | nullable; the footer's legal line — rendered as `© <year> <copyright>` (falls back to the site name when unset) |
| updated_at | timestamptz | |

> Branding must not be hardcoded. The code ships a generic default (`site_settings`), which the Super Owner overrides from the Owner Control Panel; the public main page always renders from `site_settings`. Uploaded logo/favicon/avatar files use **unique filenames** (`<id>-<nonce>.<ext>`), so re-uploads produce fresh URLs (browser-cache safe) and replaced files are deleted best-effort — the old fixed name (`logo.png`) kept the same URL and served a stale cached image.

### search_history
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| user_id | UUID (FK → users) | indexed with created_at |
| query | text | sanitized question as asked |
| session_id | UUID (FK → chat_sessions) | nullable; the chat thread the search ran in |
| message_id | UUID (FK → messages) | nullable; the assistant message that answered the search — the summary source |
| created_at | timestamptz | |

> Every non-URL ask by a signed-in user is recorded here. It powers the main-page "recent searches" chips, the AI-composed suggestions, and the `/history` page (which shows an on-demand **summary** — the linked assistant message's content, truncated — so the full answer text is never duplicated in this table; only two UUIDs per row). Clearing removes the user's rows and their cached suggestions. Deliberately separate from `usage_logs`, which keeps admin statistics and quota accounting intact.

### usage_logs
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| user_id | UUID (FK → users) | nullable for anonymous |
| query | text | sanitized |
| latency_ms | int | |
| status | enum (message_status) | queued / streaming / completed / failed |
| error_code | varchar | nullable; typed constant from `backend/internal/contracts` when `status = failed` |
| error_message | text | nullable; sanitized, human-readable (no secrets/PII) |
| provider_id | UUID (FK → ai_providers) | |
| created_at | timestamptz | |

> `usage_logs` powers the AI statistics panel (success/failure rate, error list, latency percentiles). **Privacy:** the control panel never receives verbatim queries — `topQueries` are masked (first rune + `*`s) and the trending-words view aggregates lowercase terms per time bucket (stopwords dropped), so admins see *what* is trending without any user's exact query.

### notes
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| user_id | UUID (FK → users) | indexed with created_at |
| title | varchar | note title (max 120 chars client-side) |
| content | text | note body, capped at 50k runes server-side |
| source_query | text | nullable; the search query the note was saved from ("Save summary to notes") |
| session_id | UUID (FK → chat_sessions) | nullable; the chat thread the note came from |
| created_at / updated_at | timestamptz | |

> Backs the **Notes mini-app** (top-right app drawer → `/notes`). Personal per-user data: every read/write is scoped to the owning user, and cross-user access is a 404 (`NOTE01003`). The result page's "Save summary to notes" creates a note with `title = query`, `content = answer`, and the source link.

### anonymous_usage
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| visitor_id | UUID (unique) | server-issued guest token (also set as an httpOnly cookie) |
| ip_hash | varchar(64) (unique) | SHA-256 of the proxy-verified client IP — raw IPs never stored |
| created_at / updated_at | timestamptz | |

> One row claims the single **anonymous AI-usage allowance** for a visitor token **and** an IP (both unique). A row's existence means that visitor/IP already used its one guest AI search — so clearing cookies or localStorage cannot reset the count, and only the proxy-verified IP (see `TRUSTED_PROXIES`) is hashed. Signed-in users are exempt; `mode=search` asks run no LLM and don't consume the allowance. Deliberately DB-backed so the limit survives Redis downtime. Rejection → `AISY02004` (429).

### register_page_visits
| Field | Type | Notes |
| --- | --- | --- |
| id | bigserial (PK) | |
| visitor_id | UUID (unique) | same server-issued guest token as `anonymous_usage` (httpOnly cookie / `X-Visitor-ID`) |
| ip_hash | varchar(64) | SHA-256 of the client IP — informational only, **not unique** (shared/office networks would otherwise undercount) |
| created_at / updated_at | timestamptz | |

> One row per visitor who **opened the register page** (sign-in page → Create account). The unique `visitor_id` makes replays no-ops, so the Owner Control Panel's "register-page visitors" metric reflects real, unique visitors. It feeds `GET /admin/stats/visitors` and the Overview's Unique users & visitors summary, letting the owner compare registration *interest* (register-page visitors) against *conversion* (accounts in `users`). Written best-effort via `POST /stats/register-visit`; a tracking failure never blocks the sign-in page.

### crawl_jobs
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| user_id | UUID (FK → users) | |
| url | text | |
| status | enum (crawl_status) | queued / running / completed / failed / blocked |
| created_at / finished_at | timestamptz | |

### qr_payment_codes
| Field | Type | Notes |
| --- | --- | --- |
| id | UUID (PK) | |
| user_id | UUID (FK → users) | |
| payload | text | QR payload |
| expires_at | timestamptz | short-lived |
| signature | varchar | HMAC signature (see security.md) |
| created_at | timestamptz | |

> The `qr_payment_codes` entity backs the payment flow referenced by the PRD security policy. The exact checkout UX is defined by the payment feature spec; this SDD only fixes its security properties.

## 2. Taxonomies / Enums

- **user_role:** `general_user`, `super_owner`
- **user_status:** `active`, `suspended`
- **message_role:** `system`, `user`, `assistant`
- **message_status:** `queued`, `streaming`, `completed`, `failed`
- **crawl_status:** `queued`, `running`, `completed`, `failed`, `blocked`
- **provider_type:** `ollama`, `openai_compatible`, `pollinations`, `huggingface`

## 3. Caching & Runtime Data (Redis)

- **Rate-limit counters** (ask endpoint, URL submission): `ratelimit:{scope}:{key}` (sliding window).
- **Queue state** for AI requests (pending jobs) — or kept in-process behind the worker pool (see ai-pipeline.md).