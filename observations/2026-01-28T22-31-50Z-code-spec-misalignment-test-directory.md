---
id: code-spec-misalignment-test-directory
created: 2026-01-28T22:31:50Z
type: code_spec_misalignment
severity: medium
affected_specs:
  - spec-parser
affected_files:
  - specs/spec-parser/assertions/assertions-have-tests.md
  - src/**/__tests__/*.test.js
---

# Test Directory Specification Misalignment

## Issue Description
The assertion "assertions-have-tests" specifies that implementation tests should be located in `app/**/__tests__/` directories, but all actual test files are located in `src/**/__tests__/` directories. This creates confusion about the correct location for tests.

## Evidence
From `specs/spec-parser/assertions/assertions-have-tests.md:19-20,59-60`:
```
### Type 1: Implementation Tests (JavaScript)
For testing **code behavior** - located in `app/**/__tests__/`

## Implementation Test Structure

**Location:** `app/**/__tests__/`

```
app/
├── parser/
│   ├── index.js
│   └── __tests__/
│       ├── parser.test.js
│       └── ...
```

Actual test location found:
```bash
$ find src -name "*.test.js" -type f | wc -l
      40
```

All 40 test files are in `src/**/__tests__/` directories, not `app/**/__tests__/`.

## Impact
- Developers following the spec would create tests in the wrong location
- New contributors may be confused about where to place tests
- CI/CD scripts may look for tests in the wrong directory
- Documentation is misleading about project structure

## Recommendation
Update the assertion to reflect the actual test location (`src/**/__tests__/`) or move all tests to match the specification (`app/**/__tests__/`). Since the entire codebase uses `src/`, updating the assertion is likely the better approach.