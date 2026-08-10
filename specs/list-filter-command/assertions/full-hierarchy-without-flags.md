---
id: full-hierarchy-without-flags
parent: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 1
status: done
depends-on: list-subcommand-registered
---

# `spekk list` Default Is Assertions; `--json` Outputs Flat Assertion JSON

## Description

`spekk list` with no flags defaults to assertions output — same content as
`spekk list --assertions-only`. Format flags (`--json`, `--tsv`, `--csv`, table)
change encoding only, not what records are shown.

`spekk list --json` emits the same records as the default table: all assertions
as a flat array with `type: "assertions"`. Hierarchy output (`type: "hierarchy"`)
is available via `spekk next --all`.

## Success Criteria

- `spekk list` (no flags) table output has one row per assertion with a `PARENT`
  column — not one row per spec.
- `spekk list --json` produces JSON with `"type": "assertions"` and an
  `"assertions"` array (same as `spekk list --json --assertions-only`).
- `spekk list --json` and `spekk list --json --assertions-only` produce identical
  output (the flag is now redundant with the default).
- `spekk list --tsv` and `spekk list --tsv --assertions-only` produce identical
  output.
- `spekk list --csv` and `spekk list --csv --assertions-only` produce identical
  output.
- The assertion count is the same across all four formats for the same
  (specs-dir, filter) inputs.
- Hierarchy JSON output (`"type": "hierarchy"`) is NOT available from
  `spekk list`; use `spekk next --all` for that.
