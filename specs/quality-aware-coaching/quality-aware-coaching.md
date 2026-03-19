---
id: quality-aware-coaching
created: 2026-03-19T18:00:00Z
priority: 1
---

# Quality-Aware Coaching

Coach skill that checks codebase state when writing specs to decide which quality-related assertions to include. Instead of blindly adding e2e, test-id, or API validation assertions to every spec, the coach uses deterministic checks (does `client/src/services/` exist? is `@playwright/test` installed?) combined with its own judgment to include only the assertions that are relevant.

## Problem

The 6 quality commands (`api-audit`, `generate-e2e-mocks`, `swagger-audit`, `test-plan`, `tn-services-validator`, `validate-testids`) are currently manual slash commands that run regardless of context. The coach should internalize this awareness so that specs are born with the right quality assertions — and specs that don't need them aren't bloated with irrelevant checks.

## Architecture

A single coach skill file (`specs/coach-skills-system/quality-aware-assertions-skill.md`) that:

1. **Triggers** when the coach is writing assertions for features that involve APIs, frontend pages, or Django backend work
2. **Runs deterministic checks** against the target project to understand what tooling exists:
   - `client/src/services/` exists → frontend services are connected
   - `@playwright/test` in devDependencies → e2e testing is set up
   - `server/` or Django patterns exist → backend API work
   - `client/src/components/` exists → React component work
   - OpenAPI schema tooling available → swagger validation possible
3. **Decides which assertion types to include** based on the checks:
   - Backend API + services exist → include "services match backend schema" assertion
   - Frontend components changed + playwright exists → include "e2e tests cover the feature" assertion
   - New interactive components → include "components have data-testid" assertion
   - Backend API without frontend services → skip e2e/services assertions entirely
4. **Adds quality assertions** to the spec with appropriate success criteria

## Key Principle

The coach doesn't run the quality checks — it decides whether to *write assertions* that the builder will implement and the reviewer will validate. The coach is the decision-maker about what quality work is needed.

## Success Criteria

- Coach skill exists and follows the established markdown skill pattern (triggers, workflow, validation)
- Deterministic checks are documented in the skill with clear if/then logic
- Coach naturally includes or excludes quality assertions based on codebase state
- No quality assertions are added when the codebase doesn't support them (e.g., no e2e assertion if playwright isn't installed)
