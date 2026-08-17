# Documentation

This folder is the single source of truth for the **Intellisearch** project. Documentation is split by feature/concern instead of living in one big file, so each area can be read, reviewed, and updated independently.

## Where to start

| Document | Purpose |
| --- | --- |
| [`README.md`](../README.md) | Repo onboarding: what the product is, the stack, and how to run it |
| [`PRD/`](PRD/) | **Product Requirements** — goals, personas, per-feature specs, mockups, business rules |
| [`SDD/`](SDD/) | **System Design** — architecture, data models, API surface, AI pipeline, security |
| [`tech_stack/IMPLEMENTATION_STATUS.md`](tech_stack/IMPLEMENTATION_STATUS.md) | Delivered vs. pending — the code-backed truth |
| [`tech_stack/api-contracts.md`](tech_stack/api-contracts.md) | The exact HTTP contracts (envelope, endpoints, error codes) |
| [`tech_stack/sprints/MVP_Sprint_Plan.md`](tech_stack/sprints/MVP_Sprint_Plan.md) | Milestone/sprint breakdown and Definition of Done |
| [`design-language.md`](design-language.md) | Frontend design standards (tokens, anti-patterns, a11y) |

## Product Requirements (PRD)

Split by feature:

- [PRD index](PRD/) — product overview, goals, personas, acceptance criteria
- [Main page & ask flow](PRD/main-page.md)
- [Result page](PRD/result-page.md)
- [Account settings](PRD/account-settings.md)
- [Owner Control Panel](PRD/control-panel.md)
- [Security & business rules](PRD/security.md)
- [Responsive design & theming](PRD/responsive-theming.md)
- [UI revamp proposal](PRD/UI_Revamp_Proposal.md) (visual direction)

## System Design (SDD)

Split by concern:

- [SDD index](SDD/) — stack, DDD layering, milestones
- [System architecture & request flow](SDD/architecture.md)
- [Frontend architecture](SDD/frontend-architecture.md)
- [AI pipeline (handler, queue, LLM, search, crawler)](SDD/ai-pipeline.md)
- [Data models & taxonomies](SDD/data-models.md)
- [API design overview](SDD/api-design.md)
- [Security implementation](SDD/security.md)
- [Non-functional requirements & milestones](SDD/nfr-milestones.md)
- [UI revamp SDD](SDD/UI_Revamp_SDD.md) (presentation refactor notes)

## Maintenance rule

> Whenever an API, persistence model, security rule, UI workflow, runtime dependency, or deployment behavior changes, update the relevant file under `PRD/`, `SDD/`, or `tech_stack/`. Keep `README.md` as the onboarding entry point and link to the canonical docs rather than duplicating long specs.
