---
id: query-command-select-only
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: index-command-builds-db
---

# `spekk query "<sql>"` Executes Read-Only SELECT and Returns Results in Table Format

## Description

`spekk query` takes a SQL string argument, validates it is a SELECT statement,
executes it against `.spekk/index.db`, and prints results using the same table
formatter as `spekk list`.

## Success Criteria

- `spekk query "SELECT id, status FROM assertions WHERE status = 'draft'"`
  prints a table with columns `ID` and `STATUS` (derived from SELECT aliases).
- Column names in the table header match the SELECT column names (or aliases
  when `AS` is used).
- `spekk query "INSERT INTO specs ..."` exits non-zero with an error message
  such as `error: only SELECT statements are permitted`.
- `spekk query "DROP TABLE specs"` exits non-zero with the same error.
- If `.spekk/index.db` does not exist, `spekk query` exits non-zero and
  prints a message suggesting `spekk index`.
- `spekk query` is registered in `spekk help` output.
- An empty result set (query matches no rows) prints just the header line and
  exits 0.
