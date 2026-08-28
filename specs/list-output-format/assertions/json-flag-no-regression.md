---
id: json-flag-no-regression
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
depends-on: default-table-format
---

# `--json` Flag Outputs Flat Assertion JSON

## Description

`spekk list --json` produces the same records as the default table — a flat
assertion array — encoded as JSON. It does NOT produce a hierarchy. Switching
`--json` on/off only changes encoding, not content.

`spekk next --all` remains the canonical way to get the full nested hierarchy.

## Success Criteria

- `spekk list --json` produces JSON with `"type": "assertions"` and an
  `"assertions"` array. The `"type"` field is never `"hierarchy"` from
  `spekk list`.
- `spekk list --json --assertions-only` produces identical output to
  `spekk list --json` (the flag is redundant with the default).
- `spekk list --json --status draft` produces the same filtered assertion array
  as `spekk list --status draft` (table format), just encoded as JSON.
- The assertions array is sorted by priority (ascending) then ID (alphabetical),
  consistent with the table and TSV/CSV formats.
- Existing callers that pipe `spekk list --json` into `jq` and inspect
  `.assertions[]` continue to work. Callers that expected `"type": "hierarchy"`
  must use `spekk next --all --raw` or `spekk next --all` instead.
