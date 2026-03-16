---
id: test-file-organization
created: 2026-01-21T23:30:00Z
priority: 1
status: not_started
---

# Parser Test File Organization

## What Must Be True

Parser tests must be properly organized in separate, focused test files at the correct location to avoid token limits and improve maintainability.

## Current Problem

- Tests are in wrong locations: `tests/spec-parser.test.js` and `src/__tests__/spec-parser.test.js`
- Combined file size exceeds builder agent token limits (~1700 lines)
- Tests cover different concerns but are mixed together

## Target Structure

```
src/parser/__tests__/
├── parser-basic.test.js         # Basic parsing and CLI functionality
├── frontmatter-parsing.test.js  # YAML frontmatter parsing tests  
├── field-validation.test.js     # Required fields, format validation
├── json-output.test.js          # JSON output format validation
├── priority-algorithm.test.js   # Next priority identification logic
└── folder-structure.test.js     # Directory structure validation
```

## Success Criteria

- ✅ All test files are in `src/parser/__tests__/` directory
- ✅ Tests are split into focused, single-responsibility files
- ✅ No individual test file exceeds 500 lines
- ✅ All existing test coverage is preserved
- ✅ Tests run successfully with `npm test`
- ✅ Old test files in `tests/` and `src/__tests__/` are removed
- ✅ Each test file has clear, descriptive naming

**Tests:** Manual validation (run `npm test` to verify all pass)