---
id: format-aware-empty
parent: list-output-format
created: 2026-07-13T00:00:00Z
priority: 1
status: done
---

# Empty Results Respect the Active Format Flag

## Description

When `spekk list` finds no matching assertions, it emits output in the requested format. This rule applies to an empty specs directory and to any combination of status and priority filters.

Giving a TSV consumer a JSON object (even a small one) is a contract violation.

## Success Criteria

- `spekk list --tsv` with no matching assertions outputs a tab-separated header row only, then exits 0.
- `spekk list --csv` with no matching assertions outputs a CSV header row only (CRLF terminated per RFC 4180), then exits 0.
- `spekk list` (table, default) with no matching assertions outputs a plain text notice and exits 0. The notice includes each active filter value.
- `spekk list --json` with no matching assertions outputs the usual flat list object with `type: "assertions"` and `assertions: []`, then exits 0. The array is never null or absent.
- An absent specs directory, an empty directory, and a filter with no matches all follow these rules.
- Command tests check empty output in each format and JSON results with status, priority, and combined filters.
