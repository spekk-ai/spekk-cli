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

## Investigation Needed

Run the test to identify the specific assertion failures:
```bash
npm test src/parser/__tests__/cli-reads-local-specs.test.js
```

Look for:
- Assertion errors
- Expected vs actual values
- Any environment-specific issues

## Common Issues

Possible causes:
- Test assumes certain specs exist in working directory
- Working directory changed during test cleanup
- Race condition in temp directory cleanup
- Git branch detection issues with new branch-aware parser

## Validation

```bash
npm test
# Should show: # pass 119, # fail 0
```
