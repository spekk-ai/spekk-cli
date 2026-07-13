---
id: tsv-flag
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# `--tsv` Outputs Tab-Separated Values with Lowercase Header

## Description

`spekk list --tsv` prints tab-separated output: a lowercase header on the
first line followed by one tab-separated data row per record. No space padding.
Reliable for shell pipelines (`cut -f2`, `awk '{print $3}'`, `sort -k3`).

## Success Criteria

- `spekk list --tsv` produces output where fields are separated by exactly one
  tab character (`\t`), not spaces.
- The header row is `id\tstatus\tpri\ttitle` (lowercase, same column order as
  the table format).
- No trailing tab on any line.
- Values are not quoted or escaped (TSV is not CSV — commas in titles are fine
  as-is; only tabs in values would need escaping, which spekk field values
  never contain).
- A unit test verifies the tab separation and lowercase header with a two-row
  fixture.
