---
id: format-flags-with-filter-flags
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: column-width-from-content
---

# Format Flags Work with `--status`, `--assertions-only`, and `--specs-dir`

## Description

Format flags (`--tsv`, `--json`, `--csv`, `--long`) are orthogonal to filter
flags (`--status`, `--assertions-only`, `--specs-dir`). Every combination must
be valid and produce coherent output.

## Success Criteria

- `spekk list --status draft --tsv` produces TSV output containing only draft
  records.
- `spekk list --status draft --assertions-only --tsv` produces TSV output
  containing only draft assertions with a `parent` column.
- `spekk list --assertions-only --csv` produces CSV output for all assertions
  including a `parent` column.
- `spekk list --status done --long` table output includes the `FILE` column.
- `spekk list --specs-dir <path> --tsv` uses the given specs directory and
  produces TSV output.
- No combination of format and filter flags causes a panic, unhandled error, or
  garbled output.
