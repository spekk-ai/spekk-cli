---
id: create-priority-algorithm-tests
parent: test-file-organization
created: 2026-01-21T23:35:00Z
priority: 1
status: done
---

# Priority Algorithm Tests File

## What Must Be True

A focused test file exists at `src/parser/__tests__/priority-algorithm.test.js` containing all next priority identification and algorithm tests.

## Test Coverage Required

**From `tests/spec-parser.test.js`:**
- Highest priority incomplete assertion identification
- Timestamp-based tie breaking
- Done assertion filtering
- In-progress assertion inclusion
- All-done case handling
- CLI integration with priority algorithm

**Key test sections to migrate:**
- `Next Priority Identification` describe block
  - `Priority Algorithm` tests
  - `CLI Integration` tests
- Tests for priority-based selection
- Timestamp tie-breaking logic
- Status filtering (not_started, in_progress vs done)
- Null return when all complete

## File Location

```
src/parser/__tests__/priority-algorithm.test.js
```

## Success Criteria

- ✅ File exists at correct location
- ✅ Contains complete priority selection test suite
- ✅ Tests priority ordering (1 > 2 > 3)
- ✅ Tests timestamp tie-breaking (older first)
- ✅ Tests status filtering logic
- ✅ Tests CLI integration with algorithm
- ✅ File is under 400 lines
- ✅ All tests pass when run

**Tests:** `npm test` runs this file and all tests pass