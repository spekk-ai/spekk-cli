---
id: builder-skills-system
created: 2026-03-19T18:00:00Z
priority: 1
---

# Builder Skills System

Extends the builder agent with contextual skill files that provide domain-specific implementation knowledge. Parallel to the coach skills system — markdown files with instructions that the builder loads when working on relevant assertion types.

## Problem

Quality commands like `generate-e2e-mocks`, `api-audit`, and `validate-testids --fix` contain detailed implementation patterns (Playwright route interception, faker factories, test ID naming conventions) that the builder needs when working on testing/quality assertions. Currently this knowledge lives in slash commands that can't be invoked by the builder agent.

Additionally, the `PromptResolver` hardcodes `skillsDir` to the coach skills directory for all agents, and there's no layered resolution for skills (only package-level). Both coach and builder need per-agent skills with the same layered extension pattern used for prompts.

## Architecture

### Layered Skills Resolution

Skills use the same three-layer resolution pattern as agent prompts:

| Layer | Coach Skills Path | Builder Skills Path | Purpose |
|-------|------------------|--------------------|---------|
| Package | `<spekk>/specs/coach-skills-system/` | `<spekk>/specs/builder-skills-system/` | Ships with spekk, core skills |
| Global | `~/.spekk/coach-skills/` | `~/.spekk/builder-skills/` | User's personal skills across all projects |
| Local | `.spekk/coach-skills/` | `.spekk/builder-skills/` | Project-specific skills |

Local skills with the same filename override package skills. All three paths are communicated to the agent in the activation message.

### Skills Format

Skills are **markdown files** following the same pattern as coach skills:

- Each skill defines: **triggers** (what assertion types activate it), **workflow** (step-by-step implementation instructions), **patterns** (code examples and conventions to follow)
- Agent reads relevant skill files when working on matching assertions
- **Zero infrastructure** — just markdown files + agent intelligence

### Initial Builder Skills

1. **`e2e-testing-skill.md`** — Playwright e2e tests with mocked API routes, faker factories, `mockRoute`/`mockRoutes` utility pattern. Migrated from `generate-e2e-mocks` command.

2. **`api-audit-skill.md`** — Analyzing page/component API call trees to discover endpoints for mocking. Migrated from `api-audit` command.

3. **`testid-instrumentation-skill.md`** — Adding `data-testid` attributes to React components with naming conventions. Migrated from `validate-testids --fix` mode.

### Testing the Extension System

To test project-local builder skills on a target project:

1. Create `.spekk/builder-skills/my-skill.md` in the target project
2. Run `spekk builder` — the activation message includes the local skills path
3. Builder discovers and loads the skill when assertion content matches triggers

Same pattern works for coach: `.spekk/coach-skills/my-skill.md`.

## Success Criteria

- `PromptResolver` resolves per-agent skills directories with layered resolution (package → global → local)
- Builder skills directory exists at `specs/builder-skills-system/` and ships with the package
- Builder prompt references skills and explains how to use them
- Three initial builder skills exist with complete workflow instructions migrated from slash commands
- Project-local skills in `.spekk/builder-skills/` are discoverable by the builder agent
