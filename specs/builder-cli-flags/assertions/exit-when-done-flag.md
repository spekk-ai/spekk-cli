---
id: exit-when-done-flag
parent: builder-cli-flags
created: 2026-10-28T02:06:00Z
priority: 1
status: done
branch: feature/builder-exit-when-done
---

# Continuous Builder Loops Exit When Requested

## Description

An operator can run `spekk builder --exit-when-done` or `spekk loop builder --exit-when-done` to finish a finite queue. Both commands continue to wait for new work by default.

## Success Criteria

- With `--exit-when-done`, either command exits with status 0 when `spekk next` reports `complete`, including when no work exists at startup or after the last successful build.
- Without the flag, either command keeps polling after `complete`; `none_eligible` still waits with or without the flag.
- Each continuous command reports one completion-count line on exit after completion, on Ctrl+C, and once if it starts with no work; the count includes successful builds in this run, not failed builds. The count is safe to read from the signal handler.
- `spekk builder --once` keeps the message `No assertions to work on.` when no work exists.
- During an active build, Ctrl+C reaches the child, and the parent waits for the child. An interrupted `--once` build that fails exits with status 1. A continuous builder loop keeps running after the child exits.

**Tests:** cmd/spekk/builder_loop_exit_test.go
