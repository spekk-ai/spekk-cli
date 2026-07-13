---
id: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 2
---

# SQLite Index — Constant-Time Queries Over the Spec Tree

## Overview

Build a local SQLite database from the specs directory, enabling constant-time
queries at any project scale. The Markdown files remain the source of truth;
SQLite is a read-only index rebuilt from them on demand.

The motivating problem: `spekk list` currently parses all Markdown files on
every invocation. At 100 specs / 503 assertions this is fast (~100ms), but
at 1,000+ specs it becomes perceptibly slow. More importantly, agents writing
SQL queries against structured data get precise, deterministic answers for
relationship queries (reverse deps, cross-status joins) without grep heuristics
or exhaustive JSON scans.

## Design

### New command: `spekk index`

Builds or updates `.spekk/index.db` at the repo root (adjacent to `specs/`).

- Walks `specs/` (or `--specs-dir` if provided), parses all spec and assertion
  Markdown frontmatter, and inserts records into three tables.
- Prints a summary: `Indexed 100 specs, 503 assertions.`
- Is idempotent: running twice produces the same result.
- `--force` drops and recreates all tables before inserting.

The database file is gitignored. `spekk index` adds `.spekk/index.db` to
`.gitignore` if not already present (repo root `.gitignore`).

### Schema

```sql
CREATE TABLE specs (
  id       TEXT PRIMARY KEY,
  title    TEXT,
  status   TEXT,
  priority INTEGER,
  branch   TEXT,
  file     TEXT
);

CREATE TABLE assertions (
  id        TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES specs(id),
  title     TEXT,
  status    TEXT,
  priority  INTEGER,
  branch    TEXT,
  file      TEXT
);

CREATE TABLE depends_on (
  assertion_id TEXT REFERENCES assertions(id),
  depends_on_id TEXT REFERENCES assertions(id)
);
```

One row per spec in `specs`; one row per assertion in `assertions`; one row per
`depends-on` edge in `depends_on` (sparse — only populated when the frontmatter
`depends-on` field is set).

### Auto-rebuild before `spekk next`

`spekk next` checks whether `.spekk/index.db` is stale before selecting the
next assertion. Staleness is determined by comparing the mtime of `index.db`
against the most recently modified file under `specs/`. If stale (or absent),
`spekk index` is run automatically before proceeding.

### New command: `spekk query "<sql>"`

Executes a read-only SQL SELECT against the index and prints results.

- Output follows the same format flags as `spekk list` (table default, `--json`,
  `--tsv`, `--csv`). Column names come from the SELECT column aliases.
- Only SELECT statements are permitted. Any other statement type (INSERT,
  UPDATE, DELETE, DROP, PRAGMA writes, etc.) exits non-zero with an error.
- If the index does not exist, `spekk query` prints an error and suggests
  running `spekk index` first.
- Examples:
  ```
  spekk query "SELECT id, status FROM assertions WHERE status = 'draft'"
  spekk query "SELECT a.id, d.depends_on_id FROM assertions a
               JOIN depends_on d ON a.id = d.assertion_id
               WHERE d.depends_on_id = 'oceania-pricing-tier'"
  ```

## Assertions

See `assertions/` for what must be true.

## Success Criteria

- `spekk index` creates `.spekk/index.db` with all three tables populated.
- All assertions with a `depends-on` frontmatter field have edges in the
  `depends_on` table.
- `.spekk/index.db` is gitignored.
- `spekk query "SELECT id FROM specs"` returns one row per spec.
- `spekk query` supports `--json`, `--tsv`, `--csv` format flags.
- `spekk next` auto-rebuilds the index when it is stale.
- `spekk index --force` drops and rebuilds from scratch.
