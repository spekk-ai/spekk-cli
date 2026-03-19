---
id: skill-has-deterministic-checks
parent: quality-aware-coaching
created: 2026-03-19T18:01:00Z
priority: 1
status: not_started
depends-on: quality-skill-exists
branch: feature/code-quality-qa
---

# Skill defines deterministic codebase checks

The quality-aware assertions skill includes a structured decision matrix of deterministic checks the coach runs before deciding which quality assertions to include in a spec.

## Success Criteria

- Skill documents the following checks with clear if/then logic:
  - `dir-exists: client/src/services/` → frontend services are connected, can include services validation assertions
  - `has-dependency: @playwright/test` → e2e testing is set up, can include e2e assertions
  - `dir-exists: server/` or Django patterns → backend API work, can include swagger audit assertions
  - `dir-exists: client/src/components/` → React component work, can include testid assertions
  - `file-exists: playwright.config.ts` → e2e infrastructure exists
- Skill specifies what to **skip** when checks fail (e.g., no e2e assertions if playwright isn't installed)
- Skill specifies the assertion templates to use when checks pass (clear success criteria text for each assertion type)
- Decision matrix is a simple table or list, not code — the coach agent reads and applies it using its intelligence
