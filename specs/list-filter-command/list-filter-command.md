---
id: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 1
---

# `spekk list` — Filtered Spec/Assertion Enumeration

## Overview

Add a `spekk list` subcommand that returns the full spec/assertion hierarchy
(like `spekk next --all`) but supports `--status` filtering and an
`--assertions-only` flat output mode. This enables agents and tooling to
retrieve only the records they care about, dramatically reducing output size
for status-based enumeration queries.

The motivating failure: `spekk next --all` on a large project returns 162KB / ~46K
tokens for 100+ specs and 500+ assertions. Calling it just to find "which
assertions are draft?" is wasteful and overwhelms models on enumeration tasks.
A filtered call (`spekk list --status draft --assertions-only`) reduces the
output by 10-20× for typical queries.

## Design

### Subcommand: `spekk list`

Accepts the same `--specs-dir` plumbing flag as `spekk next`.

### Flag: `--status <value>`

Filters to records matching the given status. Valid values: `not_started`,
`in_progress`, `done`, `draft`, `failed`.

Filtering semantics:
- Assertions with a different status are dropped.
- Specs with zero remaining assertions after filtering are also dropped.
- Spec-level status is NOT filtered — the spec is included if it has at least
  one assertion that matches, regardless of the rolled-up spec status.

### Flag: `--assertions-only`

Produces a flat assertion list instead of the grouped hierarchy. Output type
is `"assertions"` rather than `"hierarchy"`. Each entry includes `parent` so
callers can still group by spec if needed.

Can be combined with `--status`: `spekk list --status draft --assertions-only`
returns only draft assertions as a flat array.

Without `--status`, `--assertions-only` returns ALL assertions (useful for
getting a full flat inventory without spec grouping overhead).

### Output formats

**Hierarchy output** (default, with or without `--status`):

```json
{
  "type": "hierarchy",
  "specs": [
    {
      "id": "my-spec",
      "title": "...",
      "status": "in_progress",
      "priority": 1,
      "file": "specs/my-spec/my-spec.md",
      "assertions": [
        { "id": "...", "title": "...", "status": "draft", "priority": 1, "file": "..." }
      ]
    }
  ],
  "observations": []
}
```

**Assertions-only output** (`--assertions-only`):

```json
{
  "type": "assertions",
  "assertions": [
    {
      "id": "my-assertion",
      "title": "...",
      "status": "draft",
      "priority": 1,
      "file": "specs/my-spec/assertions/my-assertion.md",
      "parent": "my-spec"
    }
  ]
}
```

## Assertions

See `assertions/` for what must be true.

## Success Criteria

- `spekk list` runs without flags and produces the same output as
  `spekk next --all` (full hierarchy, all records).
- `spekk list --status draft` returns only specs+assertions where
  assertion status = draft; output is far smaller than `spekk next --all`.
- `spekk list --status draft --assertions-only` returns a flat JSON array of
  draft assertion objects with `parent` field.
- `spekk list --status invalid-value` exits non-zero and prints valid values.
- `spekk list` appears in `spekk help` output.
