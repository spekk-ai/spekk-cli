---
id: query-read-only-connection
parent: sqlite-index
created: 2026-07-25T12:00:00Z
priority: 1
status: done
branch: feat/sqlite-index
depends-on: query-command-select-only
---

# Query Enforces Read-Only at the Connection, Not Just by Prefix

## Description

The SELECT-only guard checks the statement's first keyword, which a statement like `WITH cte AS (...) DELETE FROM specs` slips past (it begins with `WITH`). Read-only must be enforced where it cannot be bypassed: the database connection itself. `spekk query` opens the index in read-only mode so any write is rejected by SQLite regardless of how the SQL is phrased. The keyword check remains as a fast, friendly early error.

## Success Criteria

- `RunQuery` opens the database with a read-only connection (e.g. `?mode=ro`), so writes cannot succeed on that connection.
- A write disguised behind a leading `WITH ... AS (...)` (e.g. `WITH x AS (SELECT 1) DELETE FROM specs`) does not modify the database — it is rejected.
- Ordinary SELECT and `WITH ... SELECT` (CTE) queries continue to work unchanged.
- The existing first-keyword check is retained as an early, human-readable rejection for obvious non-SELECT statements.
