# PRD — Account Settings (User)

> Part of the [PRD index](index.md).

Accessed from the **Account Settings** button in the top-right corner of the main page (Google-style). Available to every logged-in user for their own profile.

The page is **tab-based** — one tab per sub-menu:

| Tab | Contents |
| --- | --- |
| Profile | Account info (name, email) and avatar upload/change with preview |
| AI Limit & Usage | Current AI usage, remaining daily quota, and rate-limit status |
| Session | Session info and log out |

> **Responsive:** on mobile the tab bar scrolls horizontally (or collapses into a segmented control on very small screens), and tab content renders as a single stacked column — profile info, avatar, usage bar, and logout flow down the page; form fields take the full width.
>
> **Theming:** the app-wide theme toggle (Light / Dark / System) is available here and in the main page header; this page renders correctly in both themes.

## Profile (tab)

- View account info (name, email).
- **Avatar:** upload or change the avatar image (with preview and a default fallback).

## AI Limit & Usage (tab)

- See current AI usage and remaining limit for the period (e.g., questions used today vs. daily quota, plus the per-user rate limit).
- Clear explanation of what happens when the limit is reached (friendly "limit reached, try again tomorrow" message).

## Session (tab)

- Session info (e.g., last login, signed-in account) and **Log out**.

## Mockup (desktop)

```
┌──────────────────────────────────────────────────────────┐
│  ← Back                                   Account Settings│
│                                                          │
│  [ Profile ] [ AI Limit & Usage ] [ Session ]            │
│                                                          │
│         (avatar)     Name                                 │
│         [Change]     email@example.com                    │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

Switching tabs swaps the content area: **AI Limit & Usage** shows the usage bar with remaining daily quota (6 / 10 questions today); **Session** shows session info and the Log out button. On mobile the tab bar scrolls horizontally and tab content stacks into a single column.