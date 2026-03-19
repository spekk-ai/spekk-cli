---
id: reviewer-agent
created: 2026-03-19T18:00:00Z
priority: 2
---

# Reviewer Agent

A fourth agent (alongside coach/builder/observer) that validates implementation quality through a two-phase gate engine. Gates are markdown files (`.gate.md`) with deterministic preconditions and LLM judgment sections. The reviewer is read-only — it reports issues but doesn't fix them.

## Problem

Post-build quality validation is currently manual. After the builder completes an assertion, there's no automated check that the OpenAPI schema matches the implementation, that frontend services are in sync with the backend, that new components have test IDs, or that a QA test plan exists for the PR.

## Architecture

### Two-Phase Gate Engine

```
Gate Definition (.gate.md)
        │
        ▼
┌─────────────────────┐
│  Gate Engine (Node)  │
│                      │
│  Phase 1: Determin.  │──skip──▶ Report: "No .tsx files changed"
│  (file checks, diff) │
│         │ pass       │
│         ▼            │
│  Phase 2: LLM Judge  │──skip──▶ Report: "Changes are styling-only"
│  (relevance scoring) │
│         │ pass       │
│         ▼            │
│  Run Gate Command    │──────▶ Execute the quality check
└─────────────────────┘
```

### Gate File Format (`.gate.md`)

```markdown
---
id: gate-id
phase: post-build
tags: [frontend, testing]
depends-on: other-gate-id
---

# Gate Name

## Preconditions
- files-changed: "**/*.{tsx,jsx}"
- dir-exists: "src/components"

## LLM Judgment
Skip if the only JSX changes are in test files or storybook files.

## Workflow
[The actual check instructions — what the reviewer agent executes]

## On Failure
- severity: warning
- action: report
```

### Precondition Types (deterministic, milliseconds)

| Check | What it does | Example |
|-------|-------------|---------|
| files-changed | Glob against `git diff --name-only` | `"**/*.tsx"` |
| dir-exists | `fs.existsSync()` on directory | `"client/src/services"` |
| file-exists | `fs.existsSync()` on file | `"playwright.config.ts"` |
| file-not-exists | Negated file check | `".skip-review"` |
| branch-matches | Regex on current branch | `"feature/*"` |
| has-dependency | Check package.json | `"@playwright/test"` |
| command-succeeds | Exit code 0 | `"gh pr view"` |

### DAG Dependencies

Gates form a dependency graph. If an upstream gate is skipped, downstream gates auto-skip. Example: `generate-e2e-mocks` depends on `api-audit` — if api-audit is skipped, e2e mock generation is auto-skipped too.

### Layered Gate Resolution

1. `<spekk-package>/gates/` — default gates shipped with spekk
2. `~/.spekk/gates/` — global user gates
3. `.spekk/gates/` — project-local gates (override by id)

### Default Gates (shipped with package)

| Gate | Phase | Preconditions | Depends On |
|------|-------|---------------|------------|
| validate-testids | post-build | files-changed: `"**/*.{tsx,jsx}"` | — |
| test-plan | pre-merge | command-succeeds: `"gh pr view"` | — |

### Project-Local Gates (user-created)

| Gate | Phase | Preconditions | Depends On |
|------|-------|---------------|------------|
| swagger-audit | post-build | files-changed: `"**/{serializers,views,viewsets}.py"` | — |
| tn-services-validator | post-build | dir-exists: `"client/src/services"` | swagger-audit |

### CLI

```
spekk review                    # Evaluate all gates, run applicable ones
spekk review --list             # Show which gates apply (deterministic only, no LLM)
spekk review --dry-run          # Full evaluation (including LLM) but don't run
spekk review --gate <id>        # Run a specific gate only
spekk review --force <id>       # Force-run gate, skip all checks
spekk review --skip <id>        # Force-skip a gate
spekk review --no-llm           # Deterministic checks only (fast, free)
spekk review --tags frontend    # Only gates tagged with "frontend"
```

## Integration Points

- **Builder loop**: Optional `--review` flag on `spekk loop builder --review` runs gates after each build
- **Coach awareness**: Coach skill for gate visibility (`quality-aware-assertions-skill.md`)
- **PromptResolver**: Add reviewer to `src/cli/prompt-resolver.js` promptFiles array

## Success Criteria

- Gate engine parses `.gate.md` files and evaluates preconditions deterministically
- Gates form a DAG with topological sort for evaluation order
- `spekk review --list` shows applicable/skipped gates with reasons (no LLM cost)
- `spekk review --no-llm` runs deterministic-only checks
- Reviewer agent prompt exists and follows established agent patterns
- CLI registered in `bin/spekk.js` and `prompt-resolver.js`
- Layered gate resolution works (package → global → local)
- All existing tests continue to pass
