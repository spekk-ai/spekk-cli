---
id: force-flag-rebuilds-from-scratch
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 2
status: done
depends-on: index-command-builds-db
---

# `spekk index --force` Drops and Rebuilds from Scratch

## Description

`spekk index --force` drops all existing tables and rebuilds the database from
scratch. This is the escape hatch for corrupted or schema-mismatched databases.

## Success Criteria

- `spekk index --force` completes successfully even if `.spekk/index.db`
  already exists and contains data.
- After `--force`, the row count in each table matches the current state of
  `specs/` (no stale rows from a previous run).
- `spekk index --force` drops and recreates every table found in
  `sqlite_master` — not just deletes rows, and not a hardcoded list, so a
  table this binary does not know about cannot keep stale rows.
- Running `spekk index --force` on a fresh repo (no existing `index.db`)
  behaves the same as running `spekk index` without `--force`.
