# UI Revamp SDD

| Field | Value |
| --- | --- |
| Scope | Frontend presentation refactor only (public + admin panel) |
| Stack | Vue 3, TypeScript, Vite, Vue Router, Pinia, native CSS |
| Status | Approved implementation approach |

## Architecture

The revamp preserves the current route and API boundaries. Presentation is divided into reusable components and modular CSS:

```
views/                 route-level composition
components/            shared UI primitives (AppHeader, AskBox, Tabs, SourceCard)
styles/tokens.css      semantic light/dark design tokens
styles/base.css        reset, typography, accessible defaults
styles/layout.css      page shell and responsive structure
styles/components.css  reusable component surfaces and controls
styles/views.css       page-specific composition
styles/admin.css       admin panel shell, dark-first dashboard styling, admin surfaces
```

`AppHeader` owns the shared brand/navigation layout. Views pass a `compact` mode only when the result-page header needs an inline composer. `AskBox` exposes variants rather than duplicating form markup.

## Admin Panel (dark-first dashboard)

The admin control panel — mounted as a nested layout (`ControlPanel.vue` wraps every `/admin/*` route) — uses its own always-dark dashboard palette. `admin.css` redefines the semantic color tokens inside the `.admin-shell` / `.admin-login` scope, so the panel renders dark regardless of the public light/dark theme and never affects public pages.

- **Sidebar:** gradient brand block, icon-per-group accordion navigation, gradient active pill with an accent indicator bar, deep navy surface.
- **Landing (`/admin`):** consolidated dashboard — stat cards (questions today, active users, AI success rate, queue health), module tiles for every admin area, and an AI service snapshot (queue depth, in-flight, rejected, avg response, active provider).
- **Surfaces:** gradient accent stat cards, dark inputs, elevated modals with a blurred backdrop, refined tables/charts/badges/providers/forms.
- **Responsive:** persistent 248px sidebar ≥1024px; hamburger drawer <1024px; single-column stacks ≤520px.

## Data and API Boundaries

`GET /api/v1/site` remains the source for public branding. Search, profile, and usage placeholders remain explicitly unavailable until their planned endpoints are implemented. This revamp makes no persistence, service, API, or security changes. The only routing change is mounting `ControlPanel.vue` as the admin layout so every `/admin/*` page renders inside the shell.
