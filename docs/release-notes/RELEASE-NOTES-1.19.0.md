# Spekk CLI 1.19.0 — Custom Frontmatter Fields, Indexed

Projects attach their own frontmatter keys to specs and assertions to slice progress by business dimension — workflows, tags, compliance markers:

```yaml
---
id: invoice-extended-fields
parent: s1-billing
created: 2026-08-06T00:00:00Z
priority: 1
status: done
workflows: w5-billing-dispute-case
tags: [infrastructure, compliance]
---
```

Until now the parser dropped every key outside the known set, so no spekk surface could see them, and every consumer re-parsed frontmatter with its own script. Now the parser preserves them and `spekk index` stores them in a new `frontmatter_fields` table — one row per distinct value. Percent-complete per workflow becomes one query:

```bash
spekk query "SELECT f.value AS workflow, COUNT(*) AS total, SUM(a.status = 'done') AS done
  FROM assertions a
  JOIN frontmatter_fields f
    ON f.owner_type = 'assertion' AND f.owner_id = a.id AND f.key = 'workflows'
  GROUP BY f.value"
```

## The rules

- Any **top-level** key outside the known set (`id`, `parent`, `created`, `priority`, `status`, `branch`, `depends-on`, `locked-by`) is a custom field. `spekk validate` accepts it without warnings, as before.
- Three list spellings index identically: a flow sequence (`tags: [a, b]`), a bare comma-separated scalar (`workflows: a, b`), and a YAML block list (`- item` lines).
- Quoting protects commas: `note: "Hello, world"` is one value, `[a, "b, c"]` is two, and a block-list item is never re-split. Only unquoted commas split.
- Line-scanner artifacts never become rows: comments, nested-map children, empty keys, and block scalars (`key: |`) are excluded.
- A repeated value inserts once, so COUNT-style reports stay accurate.

## Also in this release

- The index schema version goes to 3. Existing databases rebuild transparently on first use — no manual migration.
- `spekk index --force` now drops every table found in `sqlite_master` instead of a hardcoded list, so a binary that predates a table can never leave stale rows behind after a forced rebuild.

## Upgrade

```bash
spekk update
```
