---
id: cross-branch-flag
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: done
---

# `--cross-branch` Flag Activates Merge-Preview Mode

## Description

`spekk show` gains a distinct `--cross-branch` flag that activates the
merge-preview mode. Without the flag, `show` behaves exactly as it does today
(current working tree only).

## Success Criteria

- `spekk show --cross-branch` activates cross-branch mode; `spekk show` without
  it is unchanged from current behavior.
- The flag is **not** `--all-branches` — that name remains exclusive to
  `spekk next` and is not added to `show`. The two concepts are kept distinct.
- An optional branch name filter/glob is accepted to exclude noisy/stale branches
  (e.g. `spekk show --cross-branch --branch-filter 'feat/*'`). The exact filter
  flag name is the builder's choice but must be documented in `spekk help`.
- The flag is documented in the `spekk` CLI help/usage output for `show`.
- Flag parsing is consistent with the existing flag-handling style used elsewhere
  in `cmd/spekk/main.go`.
