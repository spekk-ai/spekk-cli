---
id: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
---

# QA Agent

## Overview

The QA agent is the validation engine of the spec-driven system. It ensures implementations are correct, contracts are honored, and assertions remain true over time.

Unlike the builder (which focuses on making ONE assertion true), the QA agent validates the entire system: API contracts, UI constraints, e2e coverage, and cross-layer consistency.

## What It Does

### 1. Validate Source of Truth

Before other checks, ensure OpenAPI schema is accurate:
- Run `/swagger-audit` to verify Django stack (migrations, serializers, viewsets) produces correct OpenAPI
- If swagger is wrong, all downstream validation is unreliable

### 2. Validate API Contracts

Run existing validation tools:
- `/tn-services-validator` - Zod schemas match OpenAPI
- `/api-audit` - List all API calls for e2e coverage

### 3. Validate UI Layer

Check that UI enforces the same constraints as API:
- Form validation rules match OpenAPI field constraints (maxLength, required, enum)
- Input field attributes match OpenAPI (maxLength, required, pattern)
- Components have `data-testid` attributes (`/validate-testids`)

### 4. Validate E2E Coverage

Ensure assertions have executable proof:
- Run `/generate-e2e-mocks` to create/update e2e tests
- Run e2e tests to verify they pass
- Check for regressions (did this change break other assertions?)

### 5. Report Findings

Generate actionable report:
- Errors (must fix before done)
- Warnings (should fix)
- Passed checks
- Fix suggestions with file locations

## Validation Chain

```
/swagger-audit (source of truth accurate?)
      ↓
/tn-services-validator (Zod ↔ OpenAPI)
      ↓
/validate-testids (test IDs exist)
      ↓
Form validation check (constraints ↔ OpenAPI)
      ↓
Input attributes check (maxLength ↔ OpenAPI)
      ↓
/api-audit (list API calls)
      ↓
/generate-e2e-mocks + run tests
      ↓
Report
```

## When QA Agent Runs

- **After builder**: Before marking assertion `done`, builder hands off to QA
- **On demand**: Developer runs `spekk qa` to validate current state
- **In CI**: Run full QA validation on PR
- **Periodic**: Audit `done` assertions for regressions

## Integration with System

**Reads from:**
- OpenAPI schema (via `tnm-openapi` MCP tools)
- Form components (pattern matching)
- Input elements (pattern matching)
- Zod schema files
- E2E test files

**Writes to:**
- Validation report (stdout or file)
- E2E test files (via `/generate-e2e-mocks`)

**Uses tools:**
- `/swagger-audit` - Validate Django → OpenAPI accuracy
- `/tn-services-validator` - Validate Zod → OpenAPI
- `/validate-testids` - Validate test IDs exist
- `/api-audit` - List API calls
- `/generate-e2e-mocks` - Generate e2e tests
- `mcp__tnm-openapi__*` - Read OpenAPI schema

## Why QA Agent Matters

The builder makes things work. QA ensures they're correct:

1. **Catches contract drift** - API changed but frontend didn't update
2. **Catches constraint gaps** - DB has limits, UI doesn't enforce them
3. **Catches regressions** - New code broke existing assertions
4. **Provides proof** - E2E tests prove assertions are true

Without QA, "done" is just an opinion. With QA, "done" is verified.

## Prompt

See `qa-agent.prompt.md` for detailed agent instructions.

## Assertions

See `assertions/` for what must be true about QA agent behavior.
