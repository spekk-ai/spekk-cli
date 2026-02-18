---
id: orchestrates-validation-tools
parent: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
status: not_started
---

# QA Agent Orchestrates Validation Tools

## What Must Be True

QA agent must run all validation tools in the correct order and aggregate their results into a single report.

## Tools Orchestrated

| Order | Tool | Purpose |
|-------|------|---------|
| 1 | `/swagger-audit` | Verify OpenAPI matches Django implementation |
| 2 | `/tn-services-validator` | Verify Zod schemas match OpenAPI |
| 3 | `/validate-testids` | Verify components have data-testid attrs |
| 4 | `/api-audit` | List all API calls for e2e coverage |
| 5 | `/generate-e2e-mocks` | Generate/update e2e tests |

## Orchestration Rules

1. **Order matters**: Run in sequence, earlier tools validate later tools' inputs
2. **Fail fast option**: If any tool has errors, can stop early (configurable)
3. **Continue option**: Run all tools and aggregate all issues
4. **Tool availability**: If a tool is not available, log warning and skip

## Success Criteria

- QA agent runs all available validation tools
- Tools run in correct dependency order
- Results from all tools aggregated into single report
- Missing tools logged as warnings, not errors
- Each tool's output clearly labeled in report

## Example Output

```
QA Validation Report
====================

[1/5] Swagger Audit
  ✓ Passed - OpenAPI matches Django implementation

[2/5] TN Services Validator
  ⚠ 2 warnings
  - UserService.getUser: response missing 'middleName' field

[3/5] Validate TestIDs
  ✓ Passed - All components have data-testid

[4/5] API Audit
  ✓ Found 24 API calls across 12 components

[5/5] Generate E2E Mocks
  ✓ Generated 8 new test cases

Summary: 0 errors, 2 warnings, 4 passed
```
