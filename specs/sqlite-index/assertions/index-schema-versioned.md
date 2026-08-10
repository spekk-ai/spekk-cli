---
id: index-schema-versioned
parent: sqlite-index
created: 2026-07-25T12:00:00Z
priority: 1
status: done
depends-on: index-command-builds-db
---

# The Index Stamps a Schema Version So Upgrades Rebuild Automatically

## Description

`.spekk/index.db` is a derived artifact, so migration is never `ALTER TABLE` — it is detect-and-rebuild. `BuildIndex` stamps the current schema version into the database (via `PRAGMA user_version`), and readers rebuild whenever the stored version does not match the binary's expected version. This makes `spekk update` self-healing: a new binary whose schema differs from an on-disk index rebuilds it transparently instead of querying stale-shaped tables.

## Success Criteria

- A `const schemaVersion` exists in `internal/index`; `BuildIndex` sets `PRAGMA user_version = schemaVersion` on every successful build.
- A helper reports the `user_version` stored in an existing database.
- After `spekk index`, the database's `user_version` equals `schemaVersion`.
- A database whose stored version differs from `schemaVersion` (including the default `0` written by a version-unaware build) is treated as needing a rebuild.
- Rebuilding on a version mismatch drops and recreates the tables (force), so a changed schema cannot leave old-shaped tables in place.
