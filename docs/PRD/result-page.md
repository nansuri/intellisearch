# PRD — Result Page (AI Search Results)

> Part of the [PRD index](index.md).

The main use case of the platform — the app behaves like **AI-as-searchable-items**: asking a question returns a search-results-style page that combines a cited AI answer with a list of scraped web sources. The ask box stays available so the user can refine the query or ask follow-ups in context.

## Layout

- **Header (persistent):** site name/logo, a compact ask box (Google-results style), and the top-right avatar/account button.
- **Result body (below the header):**
  1. **Query header** — the question, source count, and timing (e.g., "About 12 sources · 1.2s").
  2. **AI Summary card** — the combined answer synthesized from the source pages, with inline numbered citations ([1], [2], …) that link to the matching source.
  3. **Web sources list** — numbered result cards, each with favicon, domain, title, snippet, and full URL; clicking opens the page in a new tab. Ordered by relevance; each card maps to its citation number in the summary.
  4. **Follow-up thread** — the Q&A history for this query: the original question, its summary + sources, and every follow-up with its own summary and (when new web data was fetched) its own source cards.
- **Loading state:** skeleton placeholders while scraping; the summary card and source cards appear progressively as each completes.
- **Search backend:** source listings come from a self-hosted **SearXNG** metasearch container on the same server, called backend-to-backend; the crawler fetches the top sources' page content so the summary is grounded in full text, not just snippets.

## Follow-up Questions

- The ask box on the result page accepts follow-up questions; submitting keeps the current session context (original question + prior answers) and runs in the same chat session.
- Each follow-up produces its own AI answer, appended to the thread; if new web data is needed, its source cards appear inline.
- **Suggested follow-ups:** below each summary the AI proposes 2–3 follow-up chips (e.g., "What about air freight?"); tapping a chip submits it as a follow-up.
- Rate limits and the daily question quota apply to follow-ups exactly like initial questions.
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
│  ┌─ AI Summary ────────────────────────────────────────────┐ │
│  │ The cheapest options are sea freight and surface mail. │ │
│  │ Sea freight is cheapest over 10kg [1]; Japan Post      │ │
│  │ surface mail wins for small parcels under 2kg [2].     │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  Web sources                                                 │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ [1] 🌐 japanpost.jp   Cheap Shipping Methods           │ │
│  │     Surface mail rates for small parcels               │ │
│  │     https://www.japanpost.jp/international/...         │ │
│  ├─────────────────────────────────────────────────────────┤ │
│  │ [2] 🌐 freightos.com  Sea Freight Cost Calculator      │ │
│  │     Compare sea vs air freight per kg                  │ │
│  │     https://www.freightos.com/...                      │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  Suggested follow-ups: [What about air freight?] [Size limit]│
└──────────────────────────────────────────────────────────────┘
```

## Mockup (mobile)

Single stacked column; source cards take the full width and stay ≥44px tappable; the ask box remains in the header (or as a sticky bottom bar while scrolling).