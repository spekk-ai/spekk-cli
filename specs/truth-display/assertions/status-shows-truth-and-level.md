---
id: status-shows-truth-and-level
parent: truth-display
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: truth-at-head-follows-the-evidence
---

# `spekk status` Shows the Truth of Each Assertion and the Count of Each Truth Value

## Description

`spekk status` shows the truth next to each assertion, and counts the four truth values for each spec and for the project.

## Success criteria

- Each assertion line ends with ` · <level> · <truth>`, or ` · <truth>` when the level is empty: `  1 ✅ Balance is never below zero · property · verified`.
- Each spec line ends with the count of each truth value that is more than 0 among its counted (not draft) assertions: `(3/3 assertions complete · 2 verified · 1 unchecked)`.
- "Overall Progress" has one line with all four values, also when a count is 0: `🔎 Truth: 12 verified · 1 failing · 300 unchecked · 4 unconfirmed`. The four counts add up to `Total`.
- When the latest run for HEAD was on a dirty work tree, "Overall Progress" has one more line: `⚠️ Results are from a run with uncommitted changes`.
- The counts are the same as a count of `verify_truth` for the latest run at HEAD (for checked properties) and the calculated value (for prose properties).
