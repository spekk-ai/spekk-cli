---
id: filter-function-in-parser-package
parent: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 2
status: done
---

# Filter Logic Lives in the `parser` Package

## Description

The filtering logic (by status) and the flat-assertion formatter are
implemented in `internal/parser/` rather than in `cmd/spekk/main.go`, so
they can be tested and reused by other callers (e.g. the cross-branch explorer
or the web UI).

## Success Criteria

- `internal/parser/output.go` (or a new file in `internal/parser/`) exports:
  - `FilterByStatus(result *ParseResult, status string) *ParseResult` — returns
    a new ParseResult with only assertions matching the given status, and only
    specs that have at least one such assertion.
  - `FormatAssertionsFlat(result *ParseResult) ([]byte, error)` — returns the
    `{"type": "assertions", "assertions": [...]}` JSON for the flat output mode.
- Both functions have unit tests in `internal/parser/output_test.go` (or a new
  `filter_test.go`):
  - `FilterByStatus` with a known ParseResult returns only matching assertions.
  - `FilterByStatus` excludes specs with no matching assertions.
  - `FilterByStatus` with an empty ParseResult returns an empty ParseResult.
  - `FormatAssertionsFlat` produces valid JSON with the correct `"type"` field
    and correct `"parent"` on each assertion entry.
- `cmd/spekk/main.go` calls these parser functions rather than re-implementing
  filter logic in the CLI layer.
