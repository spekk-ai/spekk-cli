---
id: index-fresh-before-read
parent: sqlite-index
created: 2026-07-25T12:00:00Z
priority: 1
status: done
depends-on: index-schema-versioned
---

# Readers Ensure the Index Is Fresh Before Querying It

## Description

Freshness is a single, centralized decision, not logic duplicated per command. `EnsureFresh(specsDir, dbPath)` rebuilds the index when it is absent, when the specs are newer than the database (mtime), or when the stored schema version does not match. Every command that reads the index calls it — in particular `spekk query`, which today serves whatever is on disk and can silently return stale results after specs change.

## Success Criteria

- `internal/index` exposes `EnsureFresh(specsDir, dbPath) (rebuilt bool, err error)` that rebuilds on any of: absent database, mtime-stale specs, or schema-version mismatch.
- On a schema-version mismatch the rebuild is a force rebuild (drop + recreate); on a plain content-staleness rebuild it need not force.
- `spekk query` calls `EnsureFresh` before executing the SELECT, so a query run after editing specs reflects the edits without a manual `spekk index`.
- `spekk next`'s existing auto-rebuild is expressed through the same `EnsureFresh` path rather than a separate inline staleness check.
- If `EnsureFresh` fails, `spekk query` surfaces the problem rather than silently querying a stale or missing database.
