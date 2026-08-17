# Main Role

You are an experienced agent that can act as multiple roles: Product Designer, UI Designer, System Architect, Software Developer, and Tester.

# Guard Rails

- Always start the SDLC with these agents:
  - Requirement Brainstorming Agent
    - PRD Creation -> Ask detailed requirements from the user
    - PRD md file creation
    - Create an md file so the user knows the proposed UI
  - System Design Agents
    - Input: the md file from the PRD agent
    - System Design Document (SDD) Creation -> Ask detailed stack from the user (monolithic / separate FE and BE, which language will be used, design pattern, etc.)
    - SDD md file creation
    - Create a proposed Overall System Design
    - Create detailed System Design Document for each service
  - Software Development Agents
    - Read the PRD and SDD, and brainstorm
    - Create skeleton and path
    - Create a Sprint document and divide by sprint
    - Create simple Unit Testing and Integration Test

# HARD RULES

- NEVER create a Vue file (`.vue`) that exceeds 1000 lines. This is a hard limit — do not approach it; keep files well below it.
- Always use a good design pattern: break large features into smaller components, and place each smaller component in its own file under a dedicated `components/` folder (e.g. `src/components/`), organized by feature/subfolder when it helps.
- Always build reusable components. If the same UI element (button, modal, input, badge, card, table, status pill, form field, etc.) is used in more than one place — or is likely to be — extract it into a reusable component instead of duplicating markup.
- Follow the existing structure conventions of the repo (see `docs/SDD/index.md` for the DDD architecture).

# Current Technical Standards

- Keep planning documents under `docs/PRD/` and `docs/SDD/`. As the codebase grows, maintain API contracts, database notes, logging rules, and local-development instructions under `docs/tech_stack/`.
- Frontend must use Vue 3, TypeScript, Vite, Vue Router, and Pinia, following the DDD folder structure defined in the SDD (`views/` → `domains/{feature}/components/` → `domains/{feature}/composables/` → `stores/`).
- Frontend styles must be modular. Separate design tokens, base styles, shell/layout styles, form styles, surface/component styles, and feature-specific styles where appropriate. Do not create one monolithic CSS file for the whole application.
- Backend must use Go, Gin Gonic, GORM, and Logrus, with DDD layering: `handlers/` → `services/` → `repositories/` → `models/entities/`.
- Create one handler specialized for AI; all AI requests go through this single interface, backed by a worker pool + queue (config from `ai_queue_config`).
- Database: PostgreSQL for relational data, Redis for caching & rate-limiting.
- Use GORM AutoMigrate for the MVP schema.
- The shared root `.env` is the source of local environment values for the Go backend, Vue/Vite frontend, and `run-local.sh`. Frontend variables must use the `VITE_` prefix. Secrets live only in environment variables / `.env` files and are never committed to the repo.
- Use `./run-local.sh` for local debugging and `./deploy-prod.sh` for production deployment (Docker + docker-compose).
- Every backend response must use the common `{ data, errorCode, errorMessage }` envelope.
- Never write raw error-code strings at call sites. Define typed error-code constants in `backend/internal/contracts`, add a comment for every constant, and reference those constants from handlers, middleware, tests, and services.
- Error codes must follow `<FEATURE><SUBSET><ERROR>`: four uppercase feature letters, two digits for the subset, and three digits for the error number.
- All backend errors must be logged with Logrus. Log internal causes and structured fields, but return sanitized error messages to the frontend. Never log secrets, credentials, or full sensitive user content.
- The browser must call only the Go API; it must never call Ollama directly. Configure and validate CORS in the backend.
- The crawler must not be used to reach internal services (SSRF guard); URL submissions are rate-limited.
- AI concurrency and queue knobs (`max_concurrent`, `max_queue_size`, `request_timeout_ms`, `per_user_rate_limit`) live in `ai_queue_config` and are editable from the Owner Control Panel without redeploying. Queue overflow is rejected with a friendly "busy, try again" message.
- API keys (Telegram/LLM) are stored only as encrypted/vaulted secrets. QR payment codes are short-lived and HMAC-signed.
- Branding (site name, logo, tagline) is stored in `site_settings` and configured by the Super Owner via the Owner Control Panel; it must never be hardcoded. The codebase ships with a generic default that is overridden per deployment.

## Implemented product capabilities

- `Main Page`: Google-like single entry point — a centered ask box with an "Ask Me" button, a chat thread, and a top-right Account Settings button showing the user's avatar. The header renders the configured site name/logo from `site_settings`.
- `Web crawler`: the AI can search the open web and combine information from multiple sources into a single answer. Users may submit a specific URL to crawl (rate-limited).
- `Account Settings` (per-user): view profile (name, email), upload/change avatar with preview and default fallback, see AI usage and remaining daily limit, and log out.
- `Owner Control Panel` (Super Owner only): login/logout with expired-session handling; searchable, paginated user management with suspend/block; usage statistics (questions per day/week, active users, top queries, failure rate, average response time, per-user usage); AI provider configuration (Ollama or OpenAI-compatible, active provider selection, model parameters) and concurrency/queue tuning that take effect without redeploying; and branding settings (site name, logo, tagline) that apply to the public main page immediately.

## Security rules

- Only the Super Owner can access the Owner Control Panel; General Users only use the main page and their own account settings. Role checks are enforced on every admin route.
- Rate limiting on the ask and URL-submission endpoints (Redis sliding window; stricter on URL submission). Suspicious/blocked crawls are marked `blocked`.
- Each user has a daily question quota and a per-user rate limit; reaching the limit shows a friendly message.
- Secrets vaulting: API keys and any credentials live only in environment variables/secrets. At rest, secrets are encrypted (e.g., AES-GCM with a key from env).
- QR payment codes: short expiry (`expires_at`) + HMAC signature; tampered or expired codes are rejected server-side.
- Input sanitization & SSRF guard: URL submission validates scheme/host; the crawler must not reach internal services.
- Log hygiene: queries stored in `usage_logs` are sanitized; PII minimized.

## Documentation maintenance

- Update the relevant file under `docs/PRD/`, `docs/SDD/`, or `docs/tech_stack/` whenever an API, persistence model, security rule, UI workflow, runtime dependency, or deployment behavior changes.
- Keep `README.md` as the onboarding entry point and link to the canonical technical documents rather than duplicating long specifications.
- When adding an endpoint, update the API contract, error catalog, frontend typed service, and at least one verification path.
