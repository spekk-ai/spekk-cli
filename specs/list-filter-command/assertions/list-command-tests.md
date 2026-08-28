---
id: list-command-tests
parent: list-filter-command
created: 2026-07-13T00:00:00Z
priority: 2
status: done
depends-on: status-flag-filters-assertions
---

# Automated Tests for `runList` Command Behaviour

## Description

`runList` in `cmd/spekk/main.go` currently has zero automated tests. The logic
for flag mutual exclusion, format dispatch, filtering, and sort order must be
verified by tests so regressions are caught at CI time rather than in review.

## Success Criteria

A `cmd/spekk/list_test.go` file (or equivalent in the same package) exists
and the following scenarios are covered:

**Mutual exclusion:**
- `runList` with `--json --tsv` writes an error to stderr, exits non-zero.
- `runList` with `--json --csv` writes an error to stderr, exits non-zero.
- `runList` with `--tsv --csv` writes an error to stderr, exits non-zero.
- `runList` with only `--json` succeeds (no conflict).

**Invalid status:**
- `runList` with `--status bogus` writes a message to stdout that contains
  "bogus" and exits non-zero.

**Format-aware empty (requires tmp specs dir):**
- `runList` with `--tsv` on an empty specs directory outputs a header row
  starting with "id" (lowercase), not `{"status"`.
- `runList` with `--csv` on an empty specs directory outputs a header row
  starting with "id", not `{"status"`.

**Sort order consistency:**
- `listRows(result, true)` (assertions) is sorted by priority ascending, then
  ID alphabetical.
- The row ordering produced by `listRows` matches the assertion ordering in
  `FormatAssertionsFlat` output for the same `ParseResult`.

**Implementation note:**  
Extract `execList(args []string, stdout, stderr io.Writer, specsDir string) int`
(or equivalent) from `runList` so that tests can call it with buffer writers
and a temp directory, asserting on return code and output content without
subprocess overhead or os.Exit side-effects.
