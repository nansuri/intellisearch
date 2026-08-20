# PRD — Result Page (AI Search Results)

> Part of the [PRD index](index.md).

The main use case of the platform — the app behaves like **AI-as-searchable-items**: asking a question returns a search-results-style page that combines a cited AI answer with a list of scraped web sources. The ask box stays available so the user can refine the query or ask follow-ups in context.

## Layout

- **Header (persistent):** site name/logo, a compact ask box (Google-results style, always starts a **new search**), and the top-right avatar/account button.
- **Result body (below the header):**
  1. **Query header** — the question, source count, and timing (e.g., "About 12 sources · 1.2s").
  2. **Result tabs** — **All** (AI overview + web sources), **AI Summary** (the follow-up conversation), **Images** (when the search returned images). Defaults to **All**.
  3. **All tab → AI overview** — at the top, a plain **collapsible** summary (expanded by default; "Hide summary"/"Show summary" toggle with a one-line preview when collapsed). Below it the **Web sources list** — numbered result cards, each with favicon, domain, title, snippet, and full URL; clicking opens the page in a new tab. Ordered by relevance; each card maps to its citation number in the summary. A border separates the AI overview from the web sources.
  4. **AI Summary tab** — the **follow-up conversation only** (the initial AI overview is not duplicated here): a composer plus the follow-up thread, each follow-up with its own summary. Below the composer, a **"Suggest follow-up questions" trigger** composes three follow-up chips on demand (rate-limited, token-saving — it never runs automatically); picking a chip sends it as a follow-up, and chips reset whenever the conversation changes.
- **Loading state:** animated research mascot while scraping; the summary card and source cards appear progressively as each completes.
- **Search backend:** source listings come from a self-hosted **SearXNG** metasearch container on the same server, called backend-to-backend; the crawler fetches the top sources' page content so the summary is grounded in full text, not just snippets.

## AI Summary tab & Follow-up Questions

- The **AI Summary tab** hosts the AI overview (default collapsed behind the envelope) and the **follow-up conversation**.
- A **follow-up composer** at the top of the thread accepts the next question; submitting keeps the current session context (original question + prior answers) and runs in the same chat session. The composer disables while a follow-up is in progress.
- The header ask box has no send-mode toggle anymore — it always starts a fresh search; follow-ups live exclusively in the AI Summary tab.
- Each follow-up produces its own AI answer, appended to the thread; if new web data is needed, its source cards appear inline.
- **Suggested follow-ups:** below each summary the AI proposes 2–3 follow-up chips (e.g., "What about air freight?"); tapping a chip submits it as a follow-up.
- Rate limits and the daily question quota apply to follow-ups exactly like initial questions.
- Web source snippets are truncated after a configurable character count (`VITE_MAX_SNIPPET_CHARS`, default 180) with a "Read more"/"Show less" toggle when truncated.
- The result page URL is shareable/deep-linkable (e.g., `/search?q=…`) for the current query.

## Empty & Error States

- **No sources found:** a friendly message with suggestions to rephrase the question or submit a specific URL to crawl.
- **Provider down / queue full:** the standard error banner with retry; the ask box remains usable.

## Mockup (desktop)

```
┌──────────────────────────────────────────────────────────────┐
│  [Site]  [ Ask anything...................... ]     [avatar] │
├──────────────────────────────────────────────────────────────┤
│  cheapest way to ship to Japan                               │
│  About 12 sources · 1.2s                                     │
│                                                              │
│  All  [ AI Summary ]  Images                                 │
│  ──────────────────────────────────────────────────────────  │
│                                                              │
│  ┌─ AI Summary (envelope) ────────────────────────────────┐ │
│  │ ✦ AI OVERVIEW                     (2 results)          │ │
│  │ The cheapest options are sea freight and surface mail… │ │
│  │ ───────────────────────────────────────────────────    │ │
│  │ Just a taste — dive in for the full picture            │ │
│  │                                      Read full summary → │
│  │ ───────────────────────────────────────────────────    │ │
│  │ Keep digging                                           │ │
│  │ Ask a follow-up below and the AI keeps building…       │ │
│  │ [ ↟ Ask a follow-up question…           ] [ Send ]      │ │
│  │                                                         │ │
│  │  Follow-up 1 …                                          │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  All tab → Web search                                        │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ [1] 🌐 japanpost.jp   Cheap Shipping Methods           │ │
│  │     Surface mail rates for small parcels               │ │
│  │     https://www.japanpost.jp/international/...         │ │
│  ├─────────────────────────────────────────────────────────┤ │
│  │ [2] 🌐 freightos.com  Sea Freight Cost Calculator      │ │
│  │     Compare sea vs air freight per kg                  │ │
│  │     https://www.freightos.com/...                      │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

## Mockup (mobile)

Single stacked column; source cards take the full width and stay ≥44px tappable; the ask box remains in the header (or as a sticky bottom bar while scrolling).