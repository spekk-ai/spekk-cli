---
id: lean-test-suite
parent: coordinator-skill
created: 2026-02-25T11:00:00Z
priority: 2
status: done
---

# Test Suite Is Lean and High-Value

Tests follow lean testing principles: one test per meaningful behavior, no redundant coverage, no tests for trivial code.

## What Must Be True

### Test Quality
- ✅ One test per meaningful behavior
- ✅ Tests validate behavior, not implementation details
- ✅ No tests for trivial code (getters, simple pass-throughs)
- ✅ No redundant test coverage
- ✅ Test setup only runs where needed

### Specific Removals
- ❌ Trivial getter tests (IDs, names, simple properties)
- ❌ Redundant integration tests (already covered by unit tests)
- ❌ Over-detailed error message tests
- ❌ Excessive edge case coverage
- ❌ Unnecessary global setup/teardown

### Expected Results
- Test suite reduced by ~280 lines (~15%)
- All meaningful behaviors still tested
- Tests run faster
- Test suite remains trustworthy

## Validation

- All meaningful behaviors still covered
- No test failures
- Tests execute faster than before
- Test suite size: ~1,590 lines (from 1,870)

**Tests:** src/coach/__tests__/coordinator.test.js, src/parser/__tests__/depends-on-validation.test.js, src/parser/__tests__/branch-validation.test.js
