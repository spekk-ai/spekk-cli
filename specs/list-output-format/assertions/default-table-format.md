---
id: default-table-format
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
---

# Default Output Is Table Format with Header

## Description

`spekk list` with no format flag prints a space-padded table to stdout. The
first line is a header row; subsequent lines are data rows. Column widths are
derived from the longest value in each column (across header and data rows),
with at least two spaces of padding between columns.

## Success Criteria

- `spekk list` (no flags) outputs a header line and one data line per record.
- The header line contains `ID`, `STATUS`, `PRI`, `TITLE` (in that order).
- Each column is space-padded so all values in the column start at the same
  character position.
- There is no trailing whitespace after the last column (`TITLE`).
- Column widths are dynamic: if the longest ID is 30 characters, the ID column
  is at least 30 characters wide (plus separator padding).
- The output is valid UTF-8 plain text (no ANSI color codes in default mode).
- A unit test in `internal/formatter/formatter_test.go` verifies column
  alignment with a two-row fixture: one short ID, one long ID.
