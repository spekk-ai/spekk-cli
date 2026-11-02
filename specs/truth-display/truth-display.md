---
id: truth-display
created: 2026-11-02T21:30:00Z
priority: 1
---

# Truth Display - Each Property Shows Its Workflow Value and Its Truth

## Problem

`spekk status`, `spekk list` and `spekk show` show only the workflow value of an assertion. `done` means that a builder said it built the property. It does not say whether a check proves it at HEAD. A reader who sees `done` and nothing more trusts it too much.

## Decisions

- Spekk always shows two things for each property, and it hides neither (William, 2026-11-01):
  - what code is supposed to exist: the workflow value;
  - how strongly it is checked: the evidence level and the truth at HEAD, for example `done · unchecked`, `done · property · verified`, `done · example · failing`.
- After migration most v1 assertions show `done · unchecked`. That is correct. Spekk does not hide `unchecked`, also when a project has no strictness.
- Strictness decides only whether CI fails. It does not change the display (`strictness-config`).

## Open choices

- **`spekk list` columns.** The table gets one `TRUTH` column directly after `STATUS`, with `<level> · <truth>`, or `<truth>` when there is no level. TSV and CSV add `truth` and `level` as the last two columns, so that a script that reads a column by position keeps working. JSON adds the keys `truth` and `level`.
- **Other branches.** In `spekk show --cross-branch`, an item that exists only on another branch has no truth at this HEAD. Its `truth` and `level` are empty, and the explorer shows `not at HEAD`.
- **Draft assertions.** Draft assertions show their truth on their own lines, as other assertions do. They stay out of the counts, as `status-counts-match-the-work` says.

## Assertions

See `assertions/` for what must be true.
