---
id: create-field-validation-tests
parent: test-file-organization
created: 2026-01-21T23:33:00Z
priority: 1
status: done
---

# Field Validation Tests File

## What Must Be True

A focused test file exists at `src/parser/__tests__/field-validation.test.js` containing all spec and assertion field validation tests.

## Test Coverage Required

**From `tests/spec-parser.test.js`:**
- Required fields validation for specs and assertions
- Field format validation (kebab-case IDs, ISO timestamps, priority values)
- Status field validation (valid/invalid values)
- Duplicate ID detection (specs and assertions)
- Parent reference validation
- Error message quality

**Key test sections to migrate:**
- `Spec Parser Field Validation` describe block
  - `Required Fields Validation` tests
  - `Field Format Validation` tests
  - `Status Field Validation` tests
  - `Duplicate ID Detection` tests
  - `Parent Reference Validation` tests
  - `Error Message Quality` tests

## File Location

```
src/parser/__tests__/field-validation.test.js
```

## Success Criteria

- ✅ File exists at correct location
- ✅ Contains comprehensive field validation test suite
- ✅ Tests all required fields for specs and assertions
- ✅ Validates format requirements (kebab-case, ISO dates, priorities)
- ✅ Tests duplicate detection logic
- ✅ Validates parent reference integrity
- ✅ Tests error message quality
- ✅ File is under 600 lines
- ✅ All tests pass when run

**Tests:** `npm test` runs this file and all tests pass