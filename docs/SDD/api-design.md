# SDD — API Design Overview (v1)

> Part of the [SDD index](index.md). The exact request/response contracts and error codes are in [`../tech_stack/api-contracts.md`](../tech_stack/api-contracts.md).

All responses use the common envelope `{ data, errorCode, errorMessage }`.

| Method | Path | Access | Description |
| --- | --- | --- | --- |
| POST | /api/v1/ask | public (rate-limited) | Submit a question |
| POST | /api/v1/ask/url | public (rate-limited) | Submit a URL to crawl |
| GET | /api/v1/me | user | Get own profile (name, email, avatar, AI usage/limit) |
| PATCH | /api/v1/me | user | Update own profile (name, email, avatar URL) |
| POST | /api/v1/me/avatar | user | Upload avatar image |
| GET | /api/v1/site | public | Get public site info (site name, logo, tagline) for the main page |
| GET | /api/v1/sessions/:id | user | Get a chat session with its messages + `search_results` (deep-linked result pages) |
| POST | /api/v1/auth/login | public | Sign-in for any active account (general users and Super Owner) — returns a signed JWT plus the safe user profile; the frontend serves one unified `/login` page for every account type |
| POST | /api/v1/auth/register | public | Creates a general-user account (name, email, password ≥ 8 chars) and returns a signed JWT plus the profile |
| POST | /api/v1/auth/logout | owner | Logout |
| GET | /api/v1/admin/users | owner | List/search users |
| PATCH | /api/v1/admin/users/:id | owner | Update role / status |
| GET | /api/v1/admin/stats | owner | Usage statistics |
| GET | /api/v1/admin/stats/trends | owner | Daily/weekly question counts for the statistics charts |
| GET | /api/v1/admin/stats/ai | owner | AI statistics: success/failure rate, error list, latency percentiles, queue health |
| GET | /api/v1/admin/ai/providers | owner | List AI providers |
| POST / PATCH | /api/v1/admin/ai/providers | owner | Add / update provider |
| GET / PATCH | /api/v1/admin/ai/queue-config | owner | Read / update concurrency & queue |
| GET | /api/v1/admin/site-settings | owner | Get branding settings |
| PATCH | /api/v1/admin/site-settings | owner | Update branding settings (site name, tagline) |
| POST | /api/v1/admin/site-settings/logo | owner | Upload site logo |
| DELETE | /api/v1/admin/site-settings/logo | owner | Remove site logo (falls back to the default mark) |