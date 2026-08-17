# PRD — Responsive Design, Dark Mode & Breakpoints

> Part of the [PRD index](index.md).

The entire app — main page, account settings, and Owner Control Panel — is mobile-friendly and responsive. The layout is mobile-first: designed for small screens first and progressively enhanced for larger ones.

## Breakpoints

| Breakpoint | Range | Layout behavior |
| --- | --- | --- |
| Mobile | < 768px | Single column; control panel sidebar collapses into a drawer |
| Tablet | 768px – 1023px | Fluid layout; control panel sidebar collapses into a drawer |
| Desktop | ≥ 1024px | Full layout; persistent control panel sidebar |

## Global Rules

- **No horizontal scrolling:** the layout reflows at every breakpoint; content never overflows the viewport width.
- **Touch targets** are at least 44×44px (buttons, avatar, hamburger icon, nav items).
- **Inputs** use a ≥16px font size so mobile browsers do not auto-zoom on focus.
- **Flexible media:** avatars, logos, and images scale fluidly; no fixed pixel widths on critical UI.
- **Safe areas:** sticky headers and footers respect device safe-area insets (notches, home indicator).
- **Theme-aware:** light and dark themes are defined by design tokens; both meet WCAG AA contrast for text and UI.

## Per-Feature Behavior

- **Main page:** the ask box spans the available width with comfortable padding; the header (site name/logo + avatar) stays on top; the chat thread fills the viewport and scrolls natively; the "Ask Me" button stays reachable without scrolling.
- **Account Settings:** tabbed sub-menus (Profile / AI Limit & Usage / Session); on mobile the tab bar scrolls horizontally (or becomes a segmented control) and tab content stacks into a single column — profile fields, avatar preview, and the usage bar take the full width and scale down gracefully.
- **Owner Control Panel:** persistent sidebar with expandable parent → child menus on desktop; on tablet/mobile it collapses into a hamburger-triggered slide-in drawer with the same accordion behavior, and the active page is shown in the header. Inside pages, wide tables reflow into stacked cards and chart/stat cards stack vertically.
- **Reusable modals:** the shared Edit and Delete-confirmation modals render as centered dialogs on desktop and near-full-width dialogs (or bottom sheets) on mobile; forms and warnings stay readable and tappable at every breakpoint.
- **Result Page:** single stacked column on mobile; the AI summary card and source cards take the full width; the header ask box stays reachable (compact, or a sticky bottom ask bar on small screens) so follow-ups remain one tap away.

## Verification

- The app is tested and usable at ≥360px (small phones) and at 768px (tablet).
- At every breakpoint: no horizontal scrolling, no overlapping or clipped controls, and all primary actions reachable.

## Dark Mode

The app supports both light and dark themes. All colors are defined by design tokens (CSS custom properties) — no hardcoded colors anywhere — so the two themes stay consistent and can be re-themed per deployment.

- **Default & override:** the theme follows the OS setting (`prefers-color-scheme`) by default; a manual toggle (**Light / Dark / System**) is available in the main page header, Account Settings, and the Owner Control Panel header, and the choice is persisted per device (e.g., localStorage).
- **No flash of wrong theme:** the active theme is applied before first paint, so the page never flashes light-on-dark or dark-on-light while loading.
- **Contrast & content:** both themes meet WCAG AA contrast for text and UI; charts, status pills, avatars, and the logo render correctly in both. The branding logo ships with a light/dark variant (or a neutral mark) so it stays legible in either theme.
- **Scope:** dark mode applies to every page — main page, Account Settings, and Owner Control Panel.