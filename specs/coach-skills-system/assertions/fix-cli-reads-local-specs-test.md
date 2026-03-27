---
id: fix-cli-reads-local-specs-test
parent: coach-skills-system
created: 2026-02-25T16:08:00Z
priority: 1
status: done
---

# Fix Failing cli-reads-local-specs Test

The test `src/parser/__tests__/cli-reads-local-specs.test.js` is failing, causing 1 test failure in the suite.

## What Must Be True

- All tests pass (119/119, not 118/119)
- Test properly validates CLI reads specs from working directory
- Test doesn't have environment dependencies

## Root Cause

The test passes in isolation but fails in the full suite - **test interference**.

**Evidence:**
```bash
# Running alone:
npm test src/parser/__tests__/cli-reads-local-specs.test.js
# Result: 123/123 pass ✅

# Running in full suite:
npm test
# Result: 122/123 pass, cli-reads-local-specs fails ❌
```

**Issue:** Another test is leaving the environment in a state that breaks this test.

## Fix Required

1. **Isolate test environment** - Ensure test doesn't depend on global state
2. **Add proper cleanup** - Reset working directory, clear temp files
3. **Fix test ordering** - Make test resilient to execution order
4. **Check beforeEach/afterEach** - Verify proper setup/teardown

## Validation

```bash
npm test
# Should show: # pass 119, # fail 0
```
