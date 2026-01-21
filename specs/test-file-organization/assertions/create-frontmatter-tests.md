---
id: create-frontmatter-tests
parent: test-file-organization
created: 2026-01-21T23:32:00Z
priority: 1
status: not_started
---

# Frontmatter Parsing Tests File

## What Must Be True

A focused test file exists at `src/parser/__tests__/frontmatter-parsing.test.js` containing all YAML frontmatter parsing functionality.

## Test Coverage Required

**From `src/__tests__/spec-parser.test.js`:**
- Well-formed YAML frontmatter parsing
- Frontmatter/content separation
- Different YAML value types handling
- Error cases (missing delimiters)
- Empty frontmatter handling
- Multi-line content handling
- Real file validation

**Key tests to migrate:**
- `YAML Frontmatter Parsing` describe block (all tests)
- `parses well-formed YAML frontmatter correctly`
- `separates frontmatter from markdown content correctly`
- `handles different YAML value types`
- `throws error for missing opening/closing frontmatter delimiter`
- `handles empty frontmatter correctly`
- `handles multi-line markdown content`
- `works with real spec and assertion files`

## File Location

```
src/parser/__tests__/frontmatter-parsing.test.js
```

## Success Criteria

- ✅ File exists at correct location
- ✅ Contains complete YAML frontmatter test suite
- ✅ Tests all parsing edge cases
- ✅ Validates real file compatibility
- ✅ File is under 400 lines
- ✅ All tests pass when run

**Tests:** `npm test` runs this file and all tests pass