---
id: runs-e2e-tests
parent: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
status: not_started
---

# QA Agent Runs E2E Tests for Coverage and Regression

## What Must Be True

QA agent must run e2e tests to verify assertions are actually true and catch regressions.

## Why E2E Tests Matter

- Unit tests verify implementation details
- E2E tests verify **user-facing behavior**
- E2E tests are **proof** that an assertion is true

Without e2e tests, "done" is just an opinion.

## What To Check

### 1. E2E Coverage

For assertions with `validation-tools` that include `generate-e2e-mocks`:
- Does an e2e test exist for this assertion?
- Does the test cover the assertion's success criteria?

### 2. E2E Test Results

- Run all e2e tests (or subset for current feature)
- All tests must pass
- Report failures with details

### 3. Regression Detection

When validating after a change:
- Run e2e tests for related assertions (not just current one)
- Flag if previously-passing tests now fail

## Workflow

```
1. Run /api-audit to identify API calls
2. Run /generate-e2e-mocks to create/update tests
3. Run e2e test suite (Playwright)
4. Report results:
   - New tests generated
   - Tests passed
   - Tests failed (with details)
   - Regressions detected
```

## Success Criteria

- QA agent generates e2e tests for assertions that need them
- Runs e2e test suite
- Reports pass/fail status for each test
- Identifies regressions (tests that passed before, fail now)
- Provides failure details (screenshots, error messages)

## Example Output

```
E2E Test Validation
===================

Generated Tests:
  + user-can-view-dashboard.spec.ts (new)
  ~ order-list-pagination.spec.ts (updated)

Test Results: 23 passed, 1 failed, 0 skipped

Failures:
  ✗ user-can-submit-order.spec.ts
    Expected: "Order submitted successfully"
    Received: "Validation error: firstName too long"
    Screenshot: .playwright/screenshots/failure-001.png

Regressions:
  ⚠ user-can-edit-profile.spec.ts
    Was passing in previous run, now fails
    Likely caused by recent changes to UserForm
```
