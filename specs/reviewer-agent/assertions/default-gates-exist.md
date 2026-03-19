---
id: default-gates-exist
parent: reviewer-agent
created: 2026-03-19T18:12:00Z
priority: 2
status: not_started
depends-on: gate-loader-parses-gate-files
branch: feature/code-quality-qa
---

# Default gate files ship with the package

Two default `.gate.md` files exist in the package `gates/` directory, providing out-of-the-box quality checks that work for any project.

## Success Criteria

- Directory `gates/` exists at the package root
- `gates/validate-testids.gate.md` exists with:
  - Preconditions: `files-changed: "**/*.{tsx,jsx}"`
  - LLM Judgment: skip if changes are styling-only or only in test/storybook files
  - Workflow: read-only validation — scan changed components for missing `data-testid` attributes and report findings
  - On Failure: severity warning, action report
- `gates/test-plan.gate.md` exists with:
  - Preconditions: `command-succeeds: "gh pr view"`, `files-changed: "**/{pages,components,templates}/**"`
  - LLM Judgment: skip if PR is trivial/docs-only
  - Workflow: generate a non-technical QA testing plan for the PR
  - On Failure: severity warning, action report
- `package.json` `files` array includes `gates/`
- Gate loader correctly discovers these files from the package path
