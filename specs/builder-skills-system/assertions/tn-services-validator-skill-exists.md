---
id: tn-services-validator-skill-exists
parent: builder-skills-system
created: 2026-03-19T19:32:00Z
priority: 2
status: draft
depends-on: builder-prompt-references-skills
---

# TN services validator skill exists with frontend/backend contract patterns

A builder skill file exists at `specs/builder-skills-system/tn-services-validator-skill.md` (or project-local `.spekk/builder-skills/tn-services-validator-skill.md`) containing the patterns for ensuring frontend TN-models service definitions match the backend OpenAPI schema.

## Success Criteria

- Has **Triggers** section: activates when assertion mentions frontend services matching backend, API contract validation, service definitions, or TN-models sync
- Has **Workflow** section covering:
  - Reading frontend service API files (`client/src/services/<service>/api.ts`) to extract baseUri, CRUD methods, custom calls
  - Reading frontend model files (`client/src/services/<service>/models.ts`) to extract Zod shapes
  - Generating/reading backend OpenAPI schema via `spectacular`
  - Comparing field-by-field: types, required, enum, maxLength, read_only/write_only
  - Checking custom service calls map to real backend endpoints
- Has **Patterns** section containing:
  - Service to backend route mapping table
  - Zod type → OpenAPI type mapping reference
  - `createApi` / `createCustomServiceCall` pattern recognition
  - Report format (endpoints, entity fields, custom calls, issues)
  - How to handle `--pr` mode (only validate changed services)
- Content migrated from the `tn-services-validator` slash command
- Project-local skill since it depends on TN-models, Django, and MCP OpenAPI tooling
