---
id: default-table-format
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
---

# Default Output Is an Assertion Table with PARENT Column

## Description

`spekk list` with no format flag prints a space-padded table of assertions to
stdout. Each row is one assertion. The `PARENT` column identifies the spec the
assertion belongs to. This makes the default content consistent with
`--json` (which also shows assertions).

## Success Criteria

- `spekk list` (no flags) outputs a header line and one data row per assertion.
- The header contains `ID`, `STATUS`, `PRI`, `PARENT`, `TITLE` (in that order).
- Each column is space-padded so all values in the column start at the same
  character position.
- There is no trailing whitespace after the last column (`TITLE`).
- Column widths are dynamic: derived from the longest value in each column
  (including the header) across all rows.
- The output is valid UTF-8 plain text (no ANSI codes in default mode).
- A unit test in `internal/formatter/formatter_test.go` or
  `cmd/spekk/list_test.go` verifies the presence of the PARENT column and that
  the row count matches the assertion count (not the spec count).
