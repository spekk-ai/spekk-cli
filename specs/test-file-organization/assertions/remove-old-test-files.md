---
id: remove-old-test-files
parent: test-file-organization
created: 2026-01-21T23:36:00Z
priority: 2
status: not_started
---

# Remove Old Test Files

## What Must Be True

Old test files in incorrect locations are removed after the new organized test structure is complete and verified.

## Files to Remove

- `tests/spec-parser.test.js` (wrong location)
- `src/__tests__/spec-parser.test.js` (wrong location)
- `src/__tests__/orchestration-loops.test.js` (if no longer needed)
- `tests/` directory (if empty after cleanup)

## Prerequisites

This assertion can only be completed after:
- All new test files are created and passing
- New test structure is verified working
- No functionality is lost in migration

## Success Criteria

- ✅ `tests/spec-parser.test.js` is deleted
- ✅ `src/__tests__/spec-parser.test.js` is deleted
- ✅ `tests/` directory removed if empty
- ✅ All tests still pass after cleanup
- ✅ No test coverage is lost
- ✅ Only `src/parser/__tests__/` contains parser tests

**Tests:** `npm test` still passes with same or better coverage

## Notes

This should be the final step after all other test organization assertions are complete.