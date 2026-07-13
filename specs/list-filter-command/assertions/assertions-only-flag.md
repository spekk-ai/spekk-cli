---
id: assertions-only-flag
parent: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: list-subcommand-registered
---

# `--assertions-only` Flag Returns a Flat Assertion List

## Description

`spekk list --assertions-only` produces a flat JSON list of assertion objects
instead of the grouped hierarchy. The `"type"` field is `"assertions"` rather
than `"hierarchy"`. Each entry includes a `"parent"` field so callers can still
group by spec.

This flag can be combined with `--status` for the most targeted output:
`spekk list --status draft --assertions-only` returns only draft assertions as
a flat array, with no spec rows mixed in.

## Success Criteria

- `spekk list --assertions-only` exits 0 and produces JSON with
  `"type": "assertions"` and an `"assertions"` array.
- Each entry in `"assertions"` has the fields:
  `id`, `title`, `status`, `priority`, `file`, `parent`.
- No `"specs"` or `"observations"` keys appear at the top level.
- `spekk list --assertions-only` (no status filter) includes ALL assertions
  from all specs.
- `spekk list --status draft --assertions-only` returns only assertions
  with `status: draft`.
- The flat list is sorted by priority (ascending) then by assertion ID
  (alphabetical) as a tiebreaker — consistent with `FormatHierarchy` sort order.
- Applying `--assertions-only` without `--status` on a repo with 100 specs and
  503 assertions produces exactly 503 entries.
