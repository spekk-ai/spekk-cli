---
id: json-flag-no-regression
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# `--json` Flag Outputs Current JSON (No Regression)

## Description

`spekk list --json` must produce byte-for-byte the same output as `spekk list`
did before this format change was introduced. The JSON format is the backwards-
compatible machine-readable mode for `jq` pipelines and any tooling that was
already parsing `spekk list` output.

## Success Criteria

- `spekk list --json` outputs a JSON object with `"type"` and either `"specs"`
  (hierarchy) or `"assertions"` (assertions-only), identical to the output that
  the previous default produced.
- `spekk list --json --assertions-only` produces the same JSON as
  `spekk list --assertions-only` did before this change.
- `spekk list --json --status draft` produces the same JSON as
  `spekk list --status draft` did before this change.
- Existing unit tests for JSON output continue to pass without modification.
