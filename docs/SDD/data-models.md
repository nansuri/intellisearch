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
| ai_daily_quota | int | daily question cap; 0 = unlimited (global default applies) |

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

### site_settings (singleton row)
| Field | Type | Notes |
| --- | --- | --- |
| id | int (PK) | single row |
| site_name | varchar | shown as logo/title on the main page; generic default shipped in code |
| logo_url | varchar | nullable; uploaded logo or default |
| tagline | varchar | nullable; optional short description |
| updated_at | timestamptz | |

> Branding must not be hardcoded. The code ships a generic default (`site_settings`), which the Super Owner overrides from the Owner Control Panel; the public main page always renders from `site_settings`.

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

> `usage_logs` powers the AI statistics panel (success/failure rate, error list, latency percentiles).

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