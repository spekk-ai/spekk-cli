---
id: index-command-builds-db
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
---

# `spekk index` Builds `.spekk/index.db` with All Three Tables Populated

## Description

Running `spekk index` in a repo with a `specs/` directory creates
`.spekk/index.db` (creating the `.spekk/` directory if needed) and populates
the `specs`, `assertions`, and `depends_on` tables from the Markdown frontmatter.

## Success Criteria

- `spekk index` exits 0 and prints a summary line such as
  `Indexed N specs, M assertions.`
- `.spekk/index.db` exists after the command completes.
- The `specs` table contains one row per spec (matching IDs from spec
  frontmatter).
- The `assertions` table contains one row per assertion; `parent_id` matches
  the parent spec's `id`.
- `spekk index` is registered in `spekk help` output.
- Running `spekk index` a second time (without `--force`) produces the same
  database contents (idempotent — upsert semantics or drop-and-recreate per
  run).
- `spekk index` accepts `--specs-dir <path>` to index a non-default location.
