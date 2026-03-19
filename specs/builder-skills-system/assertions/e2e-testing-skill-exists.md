---
id: e2e-testing-skill-exists
parent: builder-skills-system
created: 2026-03-19T18:05:00Z
priority: 1
status: draft
depends-on: builder-prompt-references-skills
---

# E2e testing skill exists with Playwright patterns

A builder skill file exists at `specs/builder-skills-system/e2e-testing-skill.md` containing the implementation patterns for creating Playwright e2e tests with mocked API routes.

## Success Criteria

- File exists at `specs/builder-skills-system/e2e-testing-skill.md`
- Has **Triggers** section: activates when assertion mentions e2e tests, Playwright, mocked API routes, or end-to-end testing
- Has **Workflow** section covering:
  - Verifying e2e infrastructure exists (playwright.config.ts, test directories)
  - Setting up infrastructure if missing (install deps, create config, create utils)
  - Running api-audit logic to discover endpoints that need mocking
  - Creating test files with proper structure
- Has **Patterns** section containing:
  - `mockRoute` / `mockRoutes` utility pattern
  - `ServerPaginatedResponseFactory` pattern
  - Entity factory pattern (e.g., `ServerUserFactory`)
  - `AuthStoreFactory` for logged-in test fixtures
  - Test naming conventions
  - Coverage checklist (happy path, error states, loading, empty, validation)
  - Third-party service mocking patterns (Stripe, PubNub, etc.)
  - snake_case convention for API mock responses
- Content migrated from the `generate-e2e-mocks` slash command
- Includes the api-audit analysis workflow (from `api-audit` command) as a sub-step for discovering what routes to intercept
