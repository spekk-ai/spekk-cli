---
id: tsv-flag
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
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
- The header row is lowercase. The v1 columns (`id`, `status`, `pri`, `parent`, `title`, and `file` with `--long`) are in table order, and `truth` and `level` are the last two columns (`truth-display`).
- No line ends with a tab, except a row whose last field (`level`) is empty.
- Values are not quoted or escaped (TSV is not CSV — commas in titles are fine
  as-is; only tabs in values would need escaping, which spekk field values
  never contain).
- A unit test in `internal/formatter/formatter_test.go` verifies the tab
  separation and lowercase header with a two-row fixture.
