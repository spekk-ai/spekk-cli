---
id: observation-custom-fields-indexed
parent: observation-index
created: 2026-08-28T20:00:00Z
priority: 1
branch: fix/observation-frontmatter-indexed
status: done
depends-on: observation-tables-schema
---

# A Custom Frontmatter Key on an Observation Reaches `frontmatter_fields`

An observation carries provenance that the closed lifecycle set has no field for — which skill found it, which run, which document narrated it. Today such a key validates, survives a round trip, and reaches no table, so it is worse than prose: a reader assumes a key is queryable. An observation gets the same custom-field treatment a spec and an assertion already get, in the same table, under the same rule.

## Success criteria

- After `spekk index`, an observation whose frontmatter carries a key outside the known lifecycle set (`slug`, `type`, `severity`, `status`, `created`, `announced`, `pr`, `affected`) has `frontmatter_fields` rows with `owner_type = 'observation'` and `owner_id` set to the slug. A known key produces no row.
- `affected` produces no `frontmatter_fields` row. It is the evidence gate, `observation_files` is its table, and a second copy of it under a different name invites a query to read the gate as a tag.
- The split rule is one rule, not a second copy of one. The same code that decides a spec's values decides an observation's, so a flow sequence (`tags: [a, b]`), a bare comma scalar (`a, b`), and a block list index identically; quoting protects commas; and comment lines, nested-map children, empty keys, and block scalars (`key: |`) never become rows.
- One row per distinct `(owner_id, key, value)`. The union holds one entry per (slug, ref) and every branch inherits every merged observation, so the same slug arrives many times; the rows are the same whether a slug is carried by one ref or by twenty. Where two refs disagree on a value, both values appear, exactly as a genuine multi-value key does.
- The index schema version rises, so an index that an earlier binary built rebuilds on first read instead of answering from rows that predate this change. The rows a build writes have changed even though no column has.
- In a repository that carries such an observation, `spekk query "SELECT owner_type, owner_id, key, value FROM frontmatter_fields WHERE owner_type = 'observation'" --tsv` returns its keys. `owner_type` now has three values, and `observation` is the one this adds. This criterion says nothing about spekk-cli's own repository, which carries no observation and no custom key, so the query is empty there and that is correct.
- `docs/cli-reference.md` names `observation` in the `owner_type` set and in the custom-field paragraph, including which observation keys are known.

**Note:** the grain is the slug, not (slug, ref), which is deliberate and is the reason for the dedup criterion above. `observations` and `observation_files` are keyed by (slug, ref) because a lifecycle field genuinely differs per ref — `status` is `resolved` on main and `open` on the branch. A custom key is not a lifecycle field, `frontmatter_fields` has no `ref` column, and issue #204 will add a fourth owner type to this table. Adding a column now for a distinction no consumer has asked for is the wrong bet. The cost is real and accepted: two refs that disagree on a key contribute both values, and the index cannot say which ref holds which.

**Tests:** `internal/observation/` — an observation file with a custom scalar, a flow sequence, and a block list exposes them as parsed fields, and no known key appears among them. `internal/index/` — a repository whose observation carries a custom key gives one row per distinct value under `owner_type = 'observation'`, two refs that agree collapse to one row while two that disagree give both values, and `affected` gives none.
