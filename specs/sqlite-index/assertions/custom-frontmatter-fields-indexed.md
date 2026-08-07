---
id: custom-frontmatter-fields-indexed
parent: sqlite-index
created: 2026-08-07T22:00:00Z
priority: 2
status: done
branch: feat/index-custom-frontmatter
depends-on: index-command-builds-db
---

# Custom Frontmatter Fields Are Indexed in `frontmatter_fields`

## Description

Projects attach their own frontmatter keys to specs and assertions (for
example `workflows: w5-patient-insurance-case` or
`tags: [infrastructure, hipaa]`) to slice progress by business dimension.
The parser preserves every key outside the known set (`id`, `parent`,
`created`, `priority`, `status`, `branch`, `depends-on`, `locked-by`) on the
parsed `Spec`/`Assertion` (`Fields`), and `spekk index` stores them in a
key-value table so `spekk query` can filter and aggregate on them:

```sql
CREATE TABLE frontmatter_fields (
  owner_type TEXT,   -- 'spec' | 'assertion'
  owner_id   TEXT,
  key        TEXT,
  value      TEXT
);
```

## Success Criteria

- After `spekk index`, a spec or assertion with a custom frontmatter key has
  one `frontmatter_fields` row per value; known keys produce no rows.
- Multi-value spellings index identically — one row per item, brackets and
  per-item quotes stripped, whitespace trimmed:
  - flow sequence: `tags: [infrastructure, hipaa]`
  - bare comma-separated scalar: `workflows: w1-a, w2-b`
  - YAML block list (`- item` lines under a bare `key:`)
- Block lists under known keys keep their previous behavior (invisible to
  the struct field); the block-list support feeds custom fields only.
- Unknown keys still pass `spekk validate` without warnings — that behavior
  is what makes the pattern possible.
- A status-by-tag report is one query:
  `SELECT f.value, COUNT(*), SUM(a.status = 'done') FROM assertions a JOIN
  frontmatter_fields f ON f.owner_type = 'assertion' AND f.owner_id = a.id
  AND f.key = 'workflows' GROUP BY f.value`.
