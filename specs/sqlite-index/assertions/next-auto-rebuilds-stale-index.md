---
id: next-auto-rebuilds-stale-index
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 2
status: done
branch: feat/list-filter-by-status
depends-on: index-command-builds-db
---

# `spekk next` Auto-Rebuilds the Index When Stale

## Description

`spekk next` checks index freshness before selecting the next assertion. If
`.spekk/index.db` is absent or older than the most recently modified file
under `specs/`, it rebuilds the index silently before proceeding.

## Success Criteria

- After modifying any file under `specs/`, running `spekk next` rebuilds
  `.spekk/index.db` (confirmed by the mtime of `index.db` updating to a time
  after the modification).
- If the index is up to date, `spekk next` does NOT rebuild it (no unnecessary
  parse overhead).
- The auto-rebuild produces no visible output to the user (silent, unless an
  error occurs).
- If the index build fails during auto-rebuild, `spekk next` exits non-zero
  with an error message.
- A unit test (or integration test) exercises the stale-index path: create
  `index.db`, touch a spec file to make it newer, run `spekk next`, verify
  `index.db` mtime updated.
