# SDD — Security Implementation

> Part of the [SDD index](index.md). The PRD defines the security policies; this doc specifies their implementation.

1. **Rate limiting** on the ask and URL-submission endpoints (Redis sliding window; stricter on URL submission). Suspicious/blocked crawls are marked `blocked`.
2. **Secrets vaulting:** Telegram API keys, LLM API keys, and any credentials live only in environment variables/secrets (never in code or repo). At rest, secrets are encrypted (e.g., AES-GCM with a key from env).
3. **QR payment codes:** short expiry (`expires_at`), signed with an HMAC (`signature`) so tampered/expired codes are rejected server-side.
4. **Auth & RBAC:** the control panel is protected by authentication (JWT sessions); role checks on every admin route (Super Owner only for AI settings and user management).
5. **Input sanitization & SSRF guard:** URL submission validates scheme/host; the crawler must not be used to reach internal services.
6. **Log hygiene:** queries stored in usage logs are sanitized; PII minimized.
7. **Search backend isolation:** SearXNG runs as an internal container reachable only inside the Docker network (never publicly exposed). Calls to it are trusted internal traffic; the SSRF guard still protects the crawler for URL submissions.
8. **Error-envelope hygiene:** the browser calls only the Go API and never third-party services directly (e.g., never Ollama); every response uses `{ data, errorCode, errorMessage }` and internal errors never leak traces.