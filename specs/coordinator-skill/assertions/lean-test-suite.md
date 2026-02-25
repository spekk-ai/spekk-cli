---
id: lean-test-suite
parent: coordinator-skill
created: 2026-02-25T11:00:00Z
priority: 2
status: not_started
---

# Test Suite Is Lean and High-Value

Tests follow lean testing principles: one test per meaningful behavior, no redundant coverage, no tests for trivial code.

## Problem

Current test suite has 1,870 lines of test code with significant bloat:

1. **Trivial getter tests** (~30 lines) - Testing constants
2. **Redundant integration tests** (~100 lines) - Already covered by unit tests
3. **Over-detailed error message tests** (~50 lines) - Testing implementation details
4. **Excessive edge case coverage** (~100 lines) - Redundant validation tests
5. **Unnecessary setup/teardown** - Full git repo for tests that don't need it

See `TEST-ISSUES.md` for detailed analysis.

## Success Criteria

### Remove Low-Value Tests

**From src/coach/__tests__/coordinator.test.js:**
- ❌ Remove: `should have correct ID` (trivial getter)
- ❌ Remove: `should have correct name` (trivial getter)
- ❌ Remove: `should have description` (trivial check)
- ❌ Remove: `should have questions` (trivial check)
- ✅ Keep: `should trigger on coordinator keywords` (actual logic)

**From src/parser/__tests__/depends-on-validation.test.js:**
- ❌ Remove: Integration test for "accepts omitted depends-on" (redundant with unit test)
- ❌ Consolidate: 5 error message tests → 1 error message quality test
- ✅ Keep: Behavior tests (rejects invalid type, format, references, etc.)

**From src/parser/__tests__/branch-validation.test.js:**
- ❌ Consolidate: 14 tests → 5 tests
  - Valid branches (multiple examples in one test)
  - Invalid characters rejected (one test, multiple examples)
  - Warnings for non-standard
  - Defaults to main
  - Integration test

### Optimize Test Setup

**From src/coach/__tests__/coordinator.test.js:**
- Move git setup from global `beforeEach()` to only tests that need it
- Use mocks for tests that just need branch name
- Reduces test execution time

### Expected Reduction

- **Before:** 1,870 lines of test code
- **After:** ~1,590 lines of test code
- **Removed:** ~280 lines of low-value tests
- **Result:** Faster tests, same behavior coverage

## Lean Testing Principles (from builder prompt)

✅ **Tests validate behavior, not implementation details**  
✅ **One test per meaningful behavior**  
✅ **Deletes redundant or low-value tests**  
✅ **No tests for trivial code** (getters, simple pass-throughs)  
✅ **Prefers integration tests over unit when appropriate** (but not redundantly)  

## Validation

- All meaningful behaviors still tested
- No test failures
- Tests run faster (less unnecessary setup)
- Test suite remains trustworthy

**Tests:** Review and refactor existing test files
