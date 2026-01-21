---
id: create-parser-basic-tests
parent: test-file-organization
created: 2026-01-21T23:31:00Z
priority: 1
status: not_started
---

# Parser Basic Tests File

## What Must Be True

A focused test file exists at `src/parser/__tests__/parser-basic.test.js` containing basic parser functionality and CLI tests.

## Test Coverage Required

**From `src/__tests__/spec-parser.test.js`:**
- Parser script existence and executability
- JSON output format validation (basic structure)
- Next priority assertion identification
- Folder structure enforcement (no flat .md files)
- Valid spec directory structure validation

**Key tests to migrate:**
- `parser script exists and is executable`
- `outputs valid JSON format`
- `identifies next priority assertion correctly`
- `enforces folder structure - no flat spec files`
- `enforces folder structure - valid spec directories`

## File Location

```
src/parser/__tests__/parser-basic.test.js
```

## Success Criteria

- ✅ File exists at correct location
- ✅ Contains all basic parser functionality tests
- ✅ Tests CLI integration properly
- ✅ Validates folder structure requirements
- ✅ File is under 300 lines
- ✅ All tests pass when run

**Tests:** `npm test` runs this file and all tests pass