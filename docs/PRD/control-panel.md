# PRD — Owner Control Panel

> Part of the [PRD index](index.md).

A brief, sidebar-based administrator panel with a hierarchical **parent → child** menu. Desktop shows a persistent left sidebar; tablet/mobile collapses it into a hamburger-triggered drawer.

## Navigation & Layout (Sidebar)

The sidebar uses a **parent → child** menu hierarchy. Parent groups are expandable/collapsible and reveal their child pages:

| Parent | Child pages |
| --- | --- |
| Dashboard | Overview (the `/admin` landing page — a consolidated snapshot of every module) |
| Users | User List, Suspended Users |
| Statistics | Overview, Top Queries, Per-User Usage, AI Stats |
| AI Settings | Providers, Queue & Concurrency |
| Branding | Site Identity, Logo |

- **Landing page (`/admin` — Dashboard / Overview):** a consolidated entry point that lists every admin module as a card (user management, suspended users, statistics, per-user usage, AI service, providers, queue & limits, site identity, logo) and shows a live snapshot: questions today, active users, AI success rate, queue health, and the active provider. Clicking a card deep-links to that module.

- **Desktop (≥1024px):** a persistent left sidebar (~240px) renders the menu; tapping a parent expands/collapses its children inline (accordion). The active child is highlighted and its parent is expanded by default on load. The selected page renders in the content area to the right.
- **Tablet / mobile (<1024px):** the sidebar is hidden by default. A hamburger icon in the panel header opens a slide-in drawer with a backdrop overlay; the same accordion behavior applies inside the drawer, and selecting a child page navigates and closes the drawer. The current page name is shown in the header so the user always knows where they are.
- **Deep links:** every child page has its own route (e.g., `/admin/statistics/top-queries`), so refreshing or sharing a URL lands on the same page.
- Inside each page, wide tables reflow into stacked cards on small screens, and chart/stat cards stack vertically.
- **Theming:** the panel header includes the app-wide theme toggle (Light / Dark / System); the panel renders correctly in both themes.

## CRUD & Reusable Modals (default)

**CRUD is the default interaction pattern for every management menu.** Any menu that manages entities (Users, AI Providers, Branding assets, etc.) supports full **Create → Read → Update → Delete** through two shared reusable components:

- **Edit modal** (used for both Create and Edit): a generic form modal with validation, loading, and error states. "Add" buttons in list headers open it empty; per-row "Edit" actions open it prefilled. Saving updates the list in place without leaving the page.
- **Delete confirmation modal**: a generic destructive-confirmation modal. Per-row "Delete" actions open it showing the entity name and a warning; deletion is executed only after the owner confirms. Cancel or closing the modal aborts.

Rules:

- The modals are built once as reusable components and shared by every menu — consistent behavior and styling (backdrop, focus trap, Escape-to-close, loading/error states). Never duplicated per menu.
- All CRUD actions require the Super Owner role; destructive actions (delete, suspend/block) always require confirmation through the delete modal.
- Exceptions to full CRUD: **Statistics** pages are read-only by nature; **singleton configs** (Queue & Concurrency, Site Identity) support read/update only; the **Logo** supports upload/replace/delete.

## Access & Authentication

- The Super Owner logs in to a separate route; the rest of the app stays public.
- Session management: login, logout, expired-session handling.

## User Management

- **List** all registered users (searchable, paginated) with per-row **Edit** and **Delete** actions.
- **Create:** add a user (name, email, role, daily quota) via the reusable edit modal.
- **Read:** view user details — email, role, status, last login.
- **Update:** edit a user's details, role, status, or quota; suspend/block users (blocks them from asking questions).
- **Delete:** remove a user after confirmation via the reusable delete modal.

## Statistics (read-only)

- **User statistics:**
  - Usage over time: questions per day/week, active users.
  - Unique user / visitor summary: registered accounts, active users, anonymous AI visitors, and unique register-page visitors (each with daily/weekly trends) — so the owner can compare registration interest against conversion.
  - Per-user usage (who asks the most, who fails the most).
- **AI statistics (detailed):**
  - **Success / failure rate** — overall and broken down by provider and model.
  - **Error list** — recent failures with typed error code, sanitized message, timestamp, provider, and query context; **filterable by error type** and counted per code.
  - **Latency** — average response time plus percentiles (p50 / p95 / p99) for successful requests.
  - **Queue health** — current queue depth, in-flight requests vs. `max_concurrent`, and the number of rejected/overflowed requests.
  - **Trends** — daily/weekly charts for the metrics above.
- Read-only by design: analytics pages expose no CRUD.

## AI Settings

- **AI Provider integration settings** — full CRUD on providers via the reusable modals:
  - Add, edit, and delete providers (e.g., local Ollama or an OpenAI-compatible provider).
  - Choose which provider is active (only one active at a time).
  - Set model parameters (e.g., temperature, max tokens).
  - API keys (if any) are stored securely.
- **Concurrent and Queue Pool** — singleton config (read/update only):
  - Max concurrent AI requests.
  - Max queue size (requests beyond this are rejected with a friendly message).
  - Request timeout.
  - Optional per-user rate limit / quota.
  - Changes take effect without redeploying the app.

## Site / Branding Settings

- Configure the public-facing identity of the main page — singleton config (read/update only via the reusable edit modal):
  - **Site name** (shown as the page logo/title).
  - **Logo / icon** (upload or replace; **delete to fall back to the default**).
  - **Tagline** (optional short description).
- Changes apply to the public main page immediately, without redeploying.

## Mockups

### Control Panel (desktop)

```
┌──────────────────────────────────────────────────────────┐
│  Owner Control Panel                         [Log out]   │
├────────────────┬─────────────────────────────────────────┤
│  ▸ Users       │  Users                    [+ Add user]  │
│  ▾ Statistics  │  [Search users.....................]   │
│      Overview  │  Name    Email       Role      Actions  │
│      Top Queries│  ─────────────────────────────────────  │
│      Per-User  │  Jane    jane@...    User      [Edit][×]│
│      AI Stats  │  ...                                    │
│  ▸ AI Settings │                                         │
│  ▸ Branding    │                                         │
└────────────────┴─────────────────────────────────────────┘
```

### Drawer (tablet / mobile)

```
┌──────────────────────────────┐
│ ☰  Owner Control Panel       │
├──────────────────────────────┤
│  Statistics / Top Queries    │
│  ...                         │
└──────────────────────────────┘

Drawer (slide-in overlay):
┌──────────────────┐
│ ▸ Users          │
│ ▾ Statistics     │
│     Overview     │
│     Top Queries  │
│     Per-User     │
│     AI Stats     │
│ ▸ AI Settings    │
│ ▸ Branding       │
└──────────────────┘
```

### Reusable modals

```
Edit modal (Create / Edit):              Delete confirmation modal:
┌───────────────────────────────┐        ┌───────────────────────────────┐
│  Edit User             [ × ]  │        │  Delete user?                 │
├───────────────────────────────┤        ├───────────────────────────────┤
│  Name      [_______________]  │        │  "jane@example.com" will be   │
│  Email     [_______________]  │        │  permanently deleted. This    │
│  Role      [ General User  ▾ ]│        │  action cannot be undone.     │
│                               │        │                               │
│             [ Cancel ] [Save] │        │         [ Cancel ] [ Delete ] │
└───────────────────────────────┘        └───────────────────────────────┘
```

On mobile both modals render as a near-full-width dialog (or bottom sheet) with the same reusable behavior: backdrop, focus trap, Escape-to-close, and loading/error states.

### AI Statistics

```
┌──────────────────────────────────────────────────────────┐
│  AI Stats                                                │
│  ┌───────────┬───────────┬───────────┬───────────────┐  │
│  │ Success   │ Failures  │ Avg resp  │ Queue: 3/20   │  │
│  │ 96.4%     │ 18 (2d)   │ 1.8s      │ in-flight 4/8 │  │
│  └───────────┴───────────┴───────────┴───────────────┘  │
│                                                          │
│  Errors (last 24h)              [filter by type ▾]      │
│  Code        Type      Count    Last seen               │
│  AISY01002  timeout     12       2 min ago               │
│  AISY02001  queue-full  5        1 hr ago                │
│  ...                                                     │
│                                                          │
│  [Latency chart: p50 1.2s · p95 3.1s · p99 6.5s]        │
└──────────────────────────────────────────────────────────┘
```

On mobile the stat cards, error table (stacked), and charts reflow into a single column.