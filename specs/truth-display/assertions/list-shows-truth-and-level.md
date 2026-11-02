---
id: list-shows-truth-and-level
parent: truth-display
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: truth-at-head-follows-the-evidence
---

# Every `spekk list` Row Shows the Truth and the Level of Its Assertion

## Description

`spekk list` shows the truth at HEAD and the evidence level next to the workflow value, in every format.

## Success criteria

- The default table has a `TRUTH` column directly after `STATUS`. Each row has `<level> · <truth>` (for example `property · verified`), or `<truth>` when the level is empty (for example `unchecked`). The cell is never empty for an assertion on this branch.
- TSV and CSV have `truth` and `level` as the last two columns. The columns of v1 keep their positions.
- `--json` adds the keys `truth` and `level` to each assertion. `level` is `""` when there is no level. No v1 key changes.
- `--long`, `--status`, `--assertions-only`, `--limit` and `--sort-by` work as before with the new column.
- Column widths, the header case and the no-trailing-whitespace rule of `list-output-format` stay true.
