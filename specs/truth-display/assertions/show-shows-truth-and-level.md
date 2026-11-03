---
id: show-shows-truth-and-level
parent: truth-display
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-truth-display
depends-on: truth-at-head-follows-the-evidence
---

# The Spec Explorer Shows the Truth and the Level of Each Assertion

## Description

`spekk show` shows both values on each assertion, and `spekk show --json` gives them to other tools.

## Success criteria

- Each assertion object in the explorer data has the keys `truth` and `level`. `spekk show --json` prints them. The v1 keys do not change.
- Each assertion in the explorer shows its workflow value and its truth together, for example `done · unchecked`. Neither is hidden, and `unchecked` has its own visible style.
- With `--cross-branch`, an item that exists only on another branch has `truth: ""` and `level: ""`, and the explorer shows `not at HEAD`.
- The order and the byte-identical-output rule of `show-order-matches-list` stay true for two runs over the same specs and the same verify results.
- Live check: in a browser, the explorer shows `done · property · verified` for a property with a passing marked check, and `done · unchecked` for a v1 assertion.

**Tests:** `cmd/spekk/truth_display_test.go`
