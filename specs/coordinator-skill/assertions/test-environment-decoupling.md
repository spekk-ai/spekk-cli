---
id: test-environment-decoupling
parent: coordinator-skill
created: 2026-02-25T11:00:00Z
priority: 1
status: done
---

# Tests Are Decoupled From Environment

All tests pass regardless of current git branch or environment state.

## What Must Be True

- ✅ All tests pass on any git branch
- ✅ Tests use mocked git branch instead of actual environment
- ✅ branch-aware-next.test.js does not depend on actual git branch
- ✅ parser-basic.test.js accounts for branch-aware filtering
- ✅ No flaky tests due to environment coupling

## Validation

```bash
# Should pass on any branch
git checkout main && npm test
git checkout feature/coordinator-skill && npm test
git checkout -b random-branch && npm test
```

**Tests:** src/parser/__tests__/branch-aware-next.test.js, src/parser/__tests__/parser-basic.test.js
