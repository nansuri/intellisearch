# Product Requirements Document — AI-Powered Jastip Platform

| Field | Value |
| --- | --- |
| **Product Name** | Intellisearch (generic — branding configurable by the owner) |
| **Document Version** | 2.9 (generic open source; no hardcoded branding) |
| **Status** | Product target; implementation status tracked separately |
| **Last Updated** | 2026-08-17 |
| **Delivery Approach** | Single-shot MVP — all features described in this document are in-scope for the initial release |
| **Related Documents** | [SDD index](../SDD/index.md) · [Implementation status](../tech_stack/IMPLEMENTATION_STATUS.md) |

> The PRD is split by feature for maintainability. This index holds the shared product context (overview, goals, personas, acceptance criteria). Per-feature specs live in the files linked below.

**Document History**

| Version | Date | Changes |
| --- | --- | --- |
| 2.1 | 2026-08-17 | Initial generic open-source release |
| 2.2 | 2026-08-17 | Owner Control Panel changed from tab-based to sidebar-based navigation; responsive / mobile-friendly requirements added |
| 2.3 | 2026-08-17 | Added app-wide dark mode; control panel sidebar upgraded to a parent → child menu hierarchy |
| 2.4 | 2026-08-17 | Account Settings converted to tab-based sub-menus, with mobile tab-bar behavior |
| 2.5 | 2026-08-17 | CRUD made the default pattern for all management menus via reusable modals |
| 2.6 | 2026-08-17 | Added Result Page — cited AI summary + scraped sources, follow-up thread, persistent ask box |
| 2.7 | 2026-08-17 | Web search sourced from self-hosted SearXNG metasearch (backend-to-backend); crawler for page content + URL submissions |
| 2.8 | 2026-08-17 | Added detailed AI statistics to the control panel |
| 2.9 | 2026-08-17 | Added implementation-status link; documented incremental delivery against the complete target scope |
| 2.10 | 2026-08-17 | Added a plain-language backend walkthrough of the ask flow with a PlantUML sequence diagram to [main-page.md](main-page.md) |

---

## 1. Product Overview

**Product Name:** Intellisearch (working title)

**Objective:** This main page is the entry point for all apps under the platform — as simple as a Google dashboard: just a text field and an "Ask Me" button. Users can ask anything from the main page. The platform is released as a generic, open-source project: no hardcoded branding — the site name, logo, and tagline are configured per deployment via the Owner Control Panel.

### 1.1 Product Goals

- **G1 — Single entry point:** every app under the platform is reachable through one simple page.
- **G2 — Answer anything:** users ask questions in natural language and get a combined answer from web search.
- **G3 — Privacy-first:** the AI stack runs locally (Ollama) with no dependency on cloud LLM providers.
- **G4 — Owner control:** the Super Owner manages users, monitors usage, and tunes AI capacity (concurrency & queue) without code changes.
- **G5 — White-label / generic:** the app ships generic; branding (site name, logo, tagline) is configured by the Super Owner, never hardcoded.

---

## 2. Target Audience & User Personas

### 2.1 General User (End User)

- **Who:** anyone using the platform — e.g., customers of a jastip (buy-for-me) service or any app on the platform.
- **Goals:** ask a question in plain language and get a quick, trustworthy answer without learning how each app works.
- **Needs:** simple UI, fast answers, visible sources to verify, account settings (avatar, AI limit), mobile-friendly.
- **Access:** main page (ask only) + own account settings. No control panel access.
- **Tech comfort:** low to medium; must not require technical knowledge.

### 2.2 Super Owner (Platform Owner)

- **Who:** the owner/administrator of the platform.
- **Goals:** oversee the platform — manage users, monitor usage, control AI infrastructure (provider, concurrency, queue).
- **Needs:** a brief admin panel: user management, usage statistics, AI settings; change provider config and concurrency limits without redeploying.
- **Access:** Owner Control Panel (full).

### 2.3 Role Summary

| Role | Main Page | Result Page | Account Settings | Control Panel: Users | Control Panel: Stats | Control Panel: AI Settings |
| --- | --- | --- | --- | --- | --- | --- |
| General User | ✅ Ask questions | ✅ Ask + follow-up | ✅ (own) | ❌ | ❌ | ❌ |
| Super Owner | ✅ | ✅ | ✅ (own) | ✅ (full) | ✅ (full) | ✅ (full) |

---

## 3. Feature Index

| Feature | File | Key sections |
| --- | --- | --- |
| Main page & ask flow (incl. web search & crawler) | [main-page.md](main-page.md) | §3.1, §4.1 |
| Account Settings | [account-settings.md](account-settings.md) | §3.2, §4.2 |
| Owner Control Panel | [control-panel.md](control-panel.md) | §3.3, §4.3–4.4, §4.6 |
| Result Page | [result-page.md](result-page.md) | §3.4, §4.5 |
| Responsive design, dark mode & breakpoints | [responsive-theming.md](responsive-theming.md) | §5 |
| Business rules & security policies | [security.md](security.md) | §6 |

---

## 4. User Stories

| ID | As a… | I want to… | So that… | Priority |
| --- | --- | --- | --- | --- |
| US-01 | General User | ask a question on the main page and get an answer | I can find anything from one place | P0 |
| US-02 | General User | submit a URL for the AI to read | it can answer about that specific page | P1 |
| US-03 | Super Owner | log in to a control panel | only I can manage the platform | P0 |
| US-04 | Super Owner | view and manage users | I can control who uses the platform | P1 |
| US-05 | Super Owner | see usage statistics | I understand how the platform is used | P1 |
| US-06 | Super Owner | change AI provider settings | I can switch models without code changes | P1 |
| US-07 | Super Owner | change concurrency & queue settings | the platform stays responsive under load | P1 |
| US-08 | Super Owner | suspend abusive users | I can protect the platform from abuse | P2 |
| US-09 | General User | open my account settings from the top-right button | I can see my profile | P1 |
| US-10 | General User | set an avatar | my account is recognizable | P2 |
| US-11 | General User | see my AI usage and remaining limit | I know how much I can still ask | P1 |
| US-12 | Super Owner | configure the site name and logo | the platform is branded for my deployment | P2 |
| US-13 | General User / Super Owner | use the app fully on my phone (main page, account settings, control panel) | I can ask questions and manage the platform on the go | P1 |
| US-14 | General User / Super Owner | use the app in dark mode (follows my device, with a manual toggle) | it's comfortable at night and matches my device | P2 |
| US-15 | Super Owner | manage any menu's data with full create/edit/delete using consistent popups | I don't have to learn a different flow for each menu | P1 |
| US-16 | General User | see the scraped web sources behind my answer | I can verify where the information comes from | P0 |
| US-17 | General User | ask follow-up questions on the result page | I can dig deeper without starting over | P1 |
| US-18 | General User | get suggested follow-up questions | I know what else I can ask | P2 |
| US-19 | Super Owner | see detailed AI statistics (success rate, error list, latency, queue health) | I can diagnose AI problems and tune capacity | P1 |

---

## 5. Acceptance Criteria

- A general user can ask a question on the main page and receive an answer.
- A user can submit a URL and get a summarized answer about that page.
- URL submission is rate-limited; abuse attempts are blocked.
- A user can open Account Settings from the top-right button and use its tabbed sub-menus (Profile, AI Limit & Usage, Session) to view their profile, set an avatar, and see their AI usage and remaining limit.
- The Super Owner can log in, list/manage users, and view statistics.
- The Super Owner can change AI provider settings and concurrency/queue values, which take effect without redeploying.
- Exceeding the queue size shows a friendly busy message instead of an error.
- The Super Owner can change the site name/logo, and the new branding appears on the public main page without redeploying.
- The main page, account settings, and Owner Control Panel are fully usable on mobile viewports (≥360px) with no horizontal scrolling.
- On desktop the control panel uses a persistent left sidebar; on tablet/mobile it collapses into a hamburger-triggered drawer.
- The control panel sidebar supports a parent → child menu: parents expand/collapse, child pages are deep-linkable, and the active child is highlighted.
- Every management menu supports Create, Read, Update, and Delete on its entities via the shared reusable Edit and Delete-confirmation modals.
- Destructive actions (delete, suspend/block) always require confirmation through the reusable delete modal.
- Asking a question opens a result page showing a cited AI summary and a list of scraped web sources (title, domain, snippet), ordered by relevance.
- The ask box remains available on the result page; users can ask follow-up questions in the same session context, and each follow-up appends its own answer and sources.
- Suggested follow-up chips appear below each summary and submit on tap.
- The app renders correctly in dark mode on all pages, with colors defined by design tokens (no hardcoded colors).
- The Owner Control Panel shows detailed AI statistics: success/failure rate (overall + per provider/model), a filterable error list, latency percentiles, and queue health.
- The theme follows the OS setting by default and can be overridden with a persistent Light / Dark / System toggle.

---

## 6. Out of Scope (MVP)

- Payments processing / checkout UX (QR codes are security-scoped only).
- Cloud LLM providers (privacy-first; local Ollama only).

---

## 7. Risks & Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Crawler abuse / scraper overload | Cost, blocklists | Rate limiting, SSRF guard |
| Ollama performance on local hardware | Slow answers | Concurrency/queue tuning, timeout handling |
| Queue overflow during spikes | User frustration | Friendly busy message, per-user limits |
| Secret leakage (Telegram/LLM keys) | Security breach | Env-only secrets, encryption at rest, no keys in repo |
| QR code forgery | Fraud | HMAC signature + short expiry |
| Single-shot MVP scope creep | Delay | Strict out-of-scope list (§6), milestone gates |
