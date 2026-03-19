---
id: builder-skills-system
created: 2026-03-19T18:00:00Z
priority: 1
---

# Builder Skills System

Extends the builder agent with contextual skill files that provide domain-specific implementation knowledge. Parallel to the coach skills system — markdown files with instructions that the builder loads when working on relevant assertion types.

## Problem

Quality commands like `generate-e2e-mocks`, `api-audit`, and `validate-testids --fix` contain detailed implementation patterns (Playwright route interception, faker factories, test ID naming conventions) that the builder needs when working on testing/quality assertions. Currently this knowledge lives in slash commands that can't be invoked by the builder agent.

## Architecture

Skills are **markdown files** in `specs/builder-skills-system/`, following the same pattern as coach skills:

- Each skill defines: **triggers** (what assertion types activate it), **workflow** (step-by-step implementation instructions), **patterns** (code examples and conventions to follow)
- Builder prompt references the skills directory
- Builder reads relevant skill files when working on matching assertions
- **Zero infrastructure** — just markdown files + builder intelligence

### Initial Skills

1. **`e2e-testing-skill.md`** — How to create Playwright e2e tests with mocked API routes, faker factories, and the `mockRoute`/`mockRoutes` utility pattern. Migrated from `generate-e2e-mocks` command.

2. **`api-audit-skill.md`** — How to analyze page/component API call trees to understand what endpoints need mocking. Migrated from `api-audit` command. Builder uses this when building e2e tests to discover what routes to intercept.

3. **`testid-instrumentation-skill.md`** — How to add `data-testid` attributes to React components following naming conventions. Migrated from `validate-testids --fix` mode. Builder uses this when building new components that need test instrumentation.

### Skill Loading

The builder prompt includes a reference to the skills directory. When working on an assertion, the builder:
1. Reads the assertion content and success criteria
2. Determines if any skills are relevant (e.g., assertion mentions e2e tests → load e2e-testing-skill)
3. Reads the relevant skill file(s)
4. Follows the skill's workflow and patterns during implementation

This is the same pattern the coach uses — no loaders, no registries, just intelligent skill selection by the agent.

## Success Criteria

- Skills directory exists at `specs/builder-skills-system/`
- Builder prompt references the skills directory and explains how to use skills
- Three initial skills exist with complete workflow instructions
- Skills contain the implementation patterns from the original slash commands
- Builder can discover and load skills contextually based on assertion content
