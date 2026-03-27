---
id: all-flag
parent: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
status: done
---

# --all Flag Is Removed (Looping Is Default)

## Description

Continuous looping is the default builder behavior. The `--all` flag is redundant and should be removed to avoid confusion.

## Success Criteria

- `spekk builder` loops through all assertions by default (no flag needed)
- Default behavior: gets next priority assertion, builds it, repeats
- Does not stop on failure — logs the failure and moves to the next assertion
- Only stops when all assertions are `done` or `failed`
- When all assertions complete, waits and polls for new work
- Ctrl+C exits gracefully
- The `--all` flag no longer exists in the CLI
- `--spec` scopes the loop to a single spec's assertions
