---
id: long-flag
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# `--long` / `-l` Flag Adds File Path Column

## Description

`--long` (alias `-l`) adds a `FILE` column to any output format. In table mode
the column appears after `TITLE`. In TSV/CSV it appears as the last field. In
JSON mode the `file` field is already present in the JSON output, so `--long`
is a no-op for `--json`.

## Success Criteria

- `spekk list --long` table output includes a `FILE` column after `TITLE`.
- `spekk list -l` (short alias) behaves identically to `--long`.
- `spekk list --long --tsv` includes a `file` field as the last tab-separated
  column; the header row gains a `file` tab at the end.
- `spekk list --long --csv` includes a `file` field as the last CSV column.
- `spekk list --long --json` produces the same output as `spekk list --json`
  (the `file` field already appears in JSON, so there is no change).
- `--long` combined with `--assertions-only` adds the file path for each
  assertion.
- A unit test in `internal/formatter/formatter_test.go` verifies the presence
  of the `FILE` column in table output and its absence without `--long`.
