---
id: observation-index
created: 2026-07-26T12:00:00Z
priority: 2
---

# Observation Index — Observations in the SQLite Index, Derived and Disposable

## Overview

Extend the SQLite spec index (`.spekk/index.db`, `internal/index`, introduced
in `specs/sqlite-index/`) to also index observations across all fetched
`observer/*` branches plus main. This gives the announce subcommand
(`specs/observer-announce/`) and any agent running `spekk query` a precise,
deterministic view of the observation landscape — which findings are open,
which are unannounced, what evidence they carry — without grep heuristics or
branch checkouts.

## The Invariant

**Every table in `.spekk/index.db` is either rebuildable from plaintext in the
repo/branch set, or safe to lose.** SQLite is strictly a derived, ephemeral
layer:

- Source of truth for observations: the markdown files on `observer/*`
  branches and main (`specs/observation-lifecycle/`).
- Deleting `.spekk/index.db` loses nothing; a rebuild reproduces it from git.
- Prompts get **SELECT-only** access via `spekk query` (already enforced by
  `specs/sqlite-index/assertions/query-command-select-only.md` and the
  read-only connection).
- All writes happen through Go code paths only — no agent, prompt, or skill
  writes to the database.

## Design Sketch

New tables (final shape is the builder's call; this is the intent):

```sql
CREATE TABLE observations (
  slug      TEXT,
  ref       TEXT,     -- branch the row was read from (observer/<slug>, main)
  type      TEXT,     -- code_spec_misalignment | outdated_specs
  severity  TEXT,     -- high | medium | low
  status    TEXT,     -- open | resolved | dismissed
  created   TEXT,
  announced TEXT,     -- NULL when frontmatter lacks announced:
  pr        TEXT,
  file      TEXT,
  PRIMARY KEY (slug, ref)
);

CREATE TABLE observation_files (
  slug TEXT,
  ref  TEXT,
  path TEXT           -- one row per affected: entry
);
```

Indexing reads observation files from refs (via the `internal/crossbranch`
read-from-ref machinery), never by checking branches out; `git fetch` is the
only remote call, consistent with the lifecycle spec's fetch-only rule.

## Assertions

See `assertions/` for what must be true.
