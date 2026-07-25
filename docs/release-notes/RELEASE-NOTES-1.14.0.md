# Spekk CLI 1.14.0 — Query Your Specs with SQL

At scale, "which draft assertions block a done one?" or "count assertions by status per spec" is awkward to answer by reading files. This release adds a SQLite index over the spec tree and a read-only SQL query interface on top of it.

## `spekk index` and `spekk query`

`spekk index` builds `.spekk/index.db` (pure-Go SQLite — no external dependency) from `specs/`, with three tables:

| Table | Columns |
|---|---|
| `specs` | `id`, `title`, `status`, `priority`, `branch`, `file` |
| `assertions` | `id`, `parent_id`, `title`, `status`, `priority`, `branch`, `file` |
| `depends_on` | `assertion_id`, `depends_on_id` |

`spekk query` runs a read-only `SELECT` against it:

```bash
spekk query "SELECT status, COUNT(*) FROM assertions GROUP BY status"
spekk query "SELECT id, title FROM specs WHERE status = 'draft'" --json
```

Output is a padded table by default, or `--json` / `--tsv` / `--csv`. Filtering, counting, grouping, and dependency joins run in SQL instead of being reconstructed by hand.

## Derived, always-fresh, and safe

- **The Markdown files remain the source of truth.** `.spekk/index.db` is a derived artifact and is added to `.gitignore` automatically.
- **Always fresh.** `spekk query` (like `spekk next`) rebuilds the index automatically when the specs have changed, so results never lag the files.
- **Self-healing across upgrades.** The index is stamped with a schema version; a spekk that expects a different schema rebuilds the index transparently on first use. Because it's derived, migration is detect-and-rebuild — never a manual step.
- **Read-only by construction.** Queries open the database read-only, so a `SELECT` can never mutate it.

## Note on dependencies

`depends_on` holds **assertion-level** edges only. Spec-level "requires/blocks" relationships live in spec prose and are intentionally not modeled as data (see the schema docs).

## Upgrade

```bash
spekk update
```
