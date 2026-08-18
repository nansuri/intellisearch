# Logging Rules

All backend request failures flow through `middleware.RespondError` (backend/internal/middleware/errors.go), which writes the standard `{ data, errorCode, errorMessage }` envelope and logs one structured line containing the `errorCode`, `route`, `method`, and `op` (a stable operation label, never user content). The internal `err` cause is attached to that log line only — never returned to the browser.

Severity is derived from the HTTP status so error logs stay meaningful:

| Status               | Level | Typical examples                                  |
| -------------------- | ----- | ------------------------------------------------- |
| 5xx                  | Error | DB/queue failures, upstream (Ollama/Pollinations) timeouts, 502/503 |
| 401, 403             | Warn  | Rejected/expired session, missing Super Owner role |
| 404, 429             | Info  | Routine miss, rate limit / daily quota reached     |
| other 4xx            | Warn  | Validation failures, malformed requests            |

Rules:
- Never log secrets, API keys, passwords, or full tokens. Log internal causes via the `error` field; send sanitized text in the envelope `message`.
- All error codes reference the typed constants in `backend/internal/contracts` — never raw strings.
- Success paths are not logged here; Gin's request logger handles access-level logs.