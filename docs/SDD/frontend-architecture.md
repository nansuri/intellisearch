# SDD — Frontend UI Architecture

> Part of the [SDD index](index.md). UI behavior is specified in the [PRD §3–§5](../PRD/index.md); this doc covers the technical choices.

## 1. Stack

- Vue 3 + TypeScript + Vite, Vue Router, Pinia (DDD folder structure: `views/` → `domains/{feature}/components/` → `domains/{feature}/composables/` → `stores/`).
- Styling via design tokens (CSS custom properties) with light + dark variants — no hardcoded colors.
- Hard rules: no `.vue` file over 1000 lines; build reusable components instead of duplicating markup.

## 2. Design Principles

- **Responsive, mobile-first:** layouts target ≥360px first and progressively enhance; no horizontal scrolling; touch targets ≥44×44px; inputs ≥16px; safe-area insets respected (PRD §5).
- **Theming via design tokens:** all colors are CSS custom properties with light + dark variants — no hardcoded colors anywhere. Theme defaults to `prefers-color-scheme` with a persisted **Light / Dark / System** toggle, applied before first paint to avoid theme flashes (PRD §5.5).
- **Reusable component library:** `BaseButton`, `BaseInput`, **`BaseModal`** (powers both the Edit modal and the Delete-confirmation modal), `Tabs`, `SidebarMenu` (parent → child accordion + mobile drawer), `ThemeToggle`, `SourceCard`, `SuggestedFollowUps`. Components are built once and shared by every view — never duplicated per feature.
- **Navigation patterns:** Account Settings uses tabbed sub-menus; the Owner Control Panel uses a persistent parent → child sidebar on desktop that collapses into a hamburger-triggered drawer on tablet/mobile, with deep-linkable child routes. `ControlPanel.vue` is mounted as a nested layout wrapping every `/admin/*` route (the admin shell was previously orphaned from the router and is now the layout for all admin pages).
- **Admin theming:** the admin panel forces its own dark-first dashboard palette by redefining the semantic tokens inside the `.admin-shell`/`.admin-login` scope (see [UI_Revamp_SDD.md](UI_Revamp_SDD.md)), so it always renders dark and never affects public pages.
- **Result Page:** a search-results-style view — cited AI summary card, numbered web-source cards from SearXNG listings, follow-up thread, suggested follow-up chips, persistent ask box — backed by `chat_sessions`, `messages`, and `search_results`.

## 3. View Map (Frontend)

| View group | Views | Notes |
| --- | --- | --- |
| Public | Main Page, Result Page | Google-like entry, ask box, chat thread |
| User | Account Settings | tabbed sub-menus: Profile · AI Limit & Usage · Session |
| Owner | Control Panel | sidebar parent → child menu, CRUD + reusable modals, statistics, AI settings, branding |