---
id: column-width-from-content
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# Table Columns Are Space-Padded to Align (Width Derived from Content)

## Description

Column widths in table output are not fixed. They are derived at render time by
scanning all rows (including the header) and taking the maximum string length
per column. This ensures the table looks correct regardless of spec/assertion
ID length distribution.

## Success Criteria

- Given two records where the first ID is 10 characters and the second is 30
  characters, both IDs are left-padded (or rather the shorter one is padded on
  the right) so that the `STATUS` column starts at the same character position
  for both rows.
- The header `ID` is also padded to match the widest ID in the data rows (not
  the other way around — data drives width).
- Minimum two spaces of separation between any two adjacent columns.
- The final column (`TITLE`, or `FILE` when `--long`) is NOT padded on the
  right — it is left-aligned and terminates at the last character.
- A unit test in `internal/formatter/formatter_test.go` verifies: given rows
  ["short", "a very long id indeed"], the second column starts at character
  position `len("a very long id indeed") + 2` for every row including the header.
