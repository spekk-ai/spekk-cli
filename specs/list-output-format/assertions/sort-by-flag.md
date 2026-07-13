---
id: sort-by-flag
parent: list-output-format
created: 2026-07-13T01:00:00Z
priority: 2
status: not_started
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# `--sort-by` Flag Sorts Output by Named Column

## Description

`spekk list --sort-by <column>` outputs assertions sorted by the specified
column in ascending order. Supported columns: `id`, `status`, `priority`,
`title`. Works with all output format flags (`--json`, `--tsv`, `--csv`,
`--long`) and filter flags (`--status`, `--priority`).

## Success Criteria

- `spekk list --sort-by id` outputs assertions sorted alphabetically by ID.
- `spekk list --sort-by priority` outputs assertions sorted by priority
  numerically ascending (priority 1 first).
- `spekk list --sort-by status` outputs assertions sorted alphabetically
  by status value.
- `spekk list --sort-by title` outputs assertions sorted alphabetically
  by title text.
- Sort is stable: assertions with equal sort keys appear in their original
  (parser) order relative to each other.
- `--sort-by` combines with `--status`, `--priority`, and all format flags.
- An unsupported column name causes a non-zero exit with a message listing
  the valid column names.
- The default output (no `--sort-by`) is unchanged (parser order).
