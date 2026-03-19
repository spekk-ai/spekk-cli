---
id: test-plan-skill-exists
parent: builder-skills-system
created: 2026-03-19T19:31:00Z
priority: 2
status: draft
depends-on: builder-prompt-references-skills
---

# Test plan skill exists with QA-friendly test plan patterns

A builder skill file exists at `specs/builder-skills-system/test-plan-skill.md` (or project-local `.spekk/builder-skills/test-plan-skill.md`) containing the patterns for writing non-technical QA testing plans that human testers can execute.

## Success Criteria

- Has **Triggers** section: activates when assertion mentions test plan, QA plan, testing plan, manual testing, or human tester verification
- Has **Workflow** section covering:
  - Analyzing what changed (categorizing into UI changes, admin changes, API-only changes)
  - Identifying test accounts from `create_test_data`
  - Determining which user roles are needed for testing
  - Writing step-by-step instructions in plain language
- Has **Patterns** section containing:
  - Default mode template (concise, PR-focused, 5-10 test items)
  - Staging mode template (comprehensive with regression, accessibility)
  - Test step format: action → expected result, always specify which user to sign in as
  - How to handle API-only changes (Swagger testing + UI smoke tests for dependent features)
  - Project-specific rules integration (`.claude/test-plan-rules.md`)
  - PR comment posting via `gh pr comment`
- Has **Deterministic checks** for conditional test plan sections:
  - `dir-exists: ios/` or `dir-exists: android/` → include native mobile manual QA steps
  - `dir-exists: client/src/` + responsive CSS patterns → include responsive viewport testing
  - No mobile indicators → skip mobile/device matrix entirely
  - `has-dependency: @playwright/test` → Playwright covers desktop browsers, don't duplicate in manual plan
  - API-only changes with no UI impact → skip browser/device matrix, focus on Swagger/contract verification
- Guidelines: plain language, no technical terms, specific button/field names, actual test account emails
- Content migrated from the `test-plan` slash command
