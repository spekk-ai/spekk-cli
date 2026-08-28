# Spekk CLI 1.24.0 — An Observation's Own Keys, Indexed

An observation records what an observer found. Its frontmatter carries the lifecycle — slug, type, severity, status, created, announced, pr, affected — and a team that wants to record anything else had two choices, both bad. Prose in the body reaches no query. A custom key looked structured, validated, and survived a round trip, and it also reached no table, which is worse: a reader assumes a key is queryable.

A spec and an assertion have carried custom keys into `frontmatter_fields` since 1.19.0. An observation now does the same.

```yaml
---
slug: parser-drops-draft-status
type: code_spec_misalignment
severity: high
status: open
created: 2026-08-28T12:00:00Z
skill: observer-prune
tags: [provenance, parser]
affected:
  - internal/parser/parser.go
---
```

```bash
spekk query "SELECT owner_id AS slug, key, value
  FROM frontmatter_fields
 WHERE owner_type = 'observation' AND key = 'skill'"
```

## The rules

- Any top-level key outside the lifecycle set (`slug`, `type`, `severity`, `status`, `created`, `announced`, `pr`, `affected`) is a custom field. The rows use `owner_type = 'observation'` and the slug as `owner_id`.
- The split rule is shared with specs and assertions, not copied. A flow sequence, a bare comma scalar, and a block list index identically; quoting protects commas; and comments, nested-map children, empty keys, and block scalars never become rows.
- `affected` stays a known key and never becomes a custom field. It is the evidence gate and the dedup key, `observation_files` is its table, and a copy of it under a custom name would invite a query to read the gate as a tag.
- The `owner_id` is the slug alone, although `observations` and `observation_files` are keyed by (slug, ref). Each observer branch is cut from main and inherits every merged observation, so the same key arrives once per branch; the rows are merged across refs, and a slug carried by twenty branches indexes exactly as a slug carried by one. Where two refs disagree on a value, both values appear.
- Ask `frontmatter_fields` alone. Joining it to `observations` on the slug returns one copy of every field row per ref, which un-merges what the table merged. Add `DISTINCT` when a join is unavoidable.

## Also in this release

- The index schema version goes to 4. No column changed; the bump is what makes an index that version 3 built rebuild, instead of answering a custom-key query from rows that predate this change. Existing databases rebuild transparently on first use — no manual migration.

## Upgrade

```bash
spekk update
```
