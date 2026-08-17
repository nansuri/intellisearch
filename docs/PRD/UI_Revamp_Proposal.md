# UI Revamp Proposal: Clean Premium Search Experience

| Field | Value |
| --- | --- |
| Scope | Current public UI: Main Page, Result Page, Account Settings |
| Status | Approved visual direction |
| Updated | 2026-08-17 |

## Design Read

The product is a consumer-facing AI research companion. It should feel calm, credible, and considered: a clean premium product interface rather than a generic search clone or a decorative "AI" experience.

## Principles

- Use cool, quiet neutral surfaces with one restrained blue accent.
- Prioritize reading and asking: generous whitespace, clear type hierarchy, and no visual clutter.
- Keep controls tactile and predictable. Primary actions are solid; secondary actions are subtle outlined surfaces.
- Use consistent 14px surfaces and pill-shaped compact controls only where they communicate an action.
- Support light and dark themes through tokens; respect reduced-motion preferences and mobile safe space.

## Page Intent

### Main Page

- A focused welcome surface with a small trust/status label, distinct product title, and a capable multi-line question composer.
- Show example prompts to help a new user start without competing with the primary task.

### Result Page

- Treat the result as a research brief: clear query context, source/time metadata, a highly readable answer surface, and a distinct source list.
- Preserve the persistent follow-up composer, with an explicit context indicator.

### Account Settings

- Present settings as a quiet account workspace: profile identity, tabs, and clearly separated preference/session actions.
- Use useful placeholder states until the corresponding API endpoints are implemented; do not imply that profile updates or metrics are live.

## Reusable UI Inventory

- `AppHeader`: responsive brand, navigation, optional compact composer.
- `AskBox`: standard and compact variants; handles submit, loading, helper text, and prompt suggestions.
- `BaseButton`: semantic action variants.
- `Tabs`, `ThemeToggle`, `SourceCard`: refined but API-independent primitives.

## Acceptance Criteria

- All existing public routes remain usable at 360px, tablet, and desktop widths.
- Light and dark themes use the same semantic tokens and meet readable contrast.
- No endpoint contract changes are required for this UI-only pass.
- Every loading, empty, and unavailable state uses clear, honest copy.
