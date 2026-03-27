---
id: tests-in-correct-location
parent: spec-parser
created: 2026-01-21T19:45:00Z
priority: 1
status: done
depends-on: target-directory-structure
branch: feature/spec-parser
---

# Tests Must Be In Correct Location

## Requirement

Spec parser tests must follow the target directory structure with tests co-located with code in `__tests__/` subdirectories.

## Success Criteria

### Correct Test Location
- ✅ Tests exist at: `src/parser/__tests__/spec-parser.test.js`
- ✅ No tests exist at: `tests/spec-parser.test.js` (wrong location)
- ✅ No tests exist at: `src/__tests__/spec-parser.test.js` (wrong location)

### Directory Structure Compliance
- ✅ Follows Node.js convention: code and tests co-located
- ✅ Parser tests live with parser code in `src/parser/__tests__/`
- ✅ Matches target directory structure spec

### Test Content
- ✅ Tests cover all parser functionality
- ✅ Tests validate JSON output format
- ✅ Tests check field validation
- ✅ Tests verify priority sorting
- ✅ Tests run with `npm run test:impl`

## Cleanup Required

**Files to remove:**
- `tests/spec-parser.test.js` (duplicate in wrong location)
- `src/__tests__/spec-parser.test.js` (duplicate in wrong location)

**Files to keep/create:**
- `src/parser/__tests__/spec-parser.test.js` (correct location)

## Rationale

The target directory structure spec clearly states tests should be co-located with code in `__tests__/` subdirectories within each module. This prevents confusion and follows Node.js conventions.

**Tests:** `src/parser/__tests__/spec-parser.test.js`