---
id: test-fixtures-use-isolated-temp-dirs
parent: optimize-test-performance
created: 2026-03-16T20:00:00Z
priority: 1
status: done
depends-on: enable-parallel-test-execution
---

# Test Fixtures Use Isolated Temp Directories

## What Must Be True

No test creates artifacts inside the project's `specs/` directory. All test fixtures live in unique directories under `os.tmpdir()`, ensuring full isolation during parallel execution.

## Success Criteria

- No test file references `path.join(process.cwd(), 'specs', 'temp-*')` or similar patterns that write into the project tree
- All test fixtures are created in `os.tmpdir()` (or equivalent) with unique per-test directory names
- All previously-skipped test suites are re-enabled and passing:
  - `src/parser/__tests__/cli-reads-local-specs.test.js` (5 `test.skip` calls)
  - `src/parser/__tests__/malformed-file-handling.test.js` (`describe.skip`)
  - `src/parser/__tests__/parser-basic.test.js` (`describe.skip`)
  - `src/parser/__tests__/priority-algorithm.test.js` (`describe.skip`)
  - `src/parser/__tests__/quote-handling.test.js` (`describe.skip`)
- `npm test` reports 0 skipped tests
- All tests pass when run in parallel (default Node.js test runner behavior)

## Context

See GitHub issue #29. Tests currently create `temp-*` directories in the shared `specs/` directory, causing cross-contamination during parallel runs. This is why 5 suites were skipped.
