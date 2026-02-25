---
id: once-flag
parent: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
status: done
---

# --once Flag Builds Single Assertion Then Exits

## Description

The default `spekk builder` behavior is to loop continuously through all assertions until complete (Ctrl+C to exit). The `--once` flag overrides this to build exactly one assertion and exit.

## Success Criteria

- `spekk builder` with no flags loops continuously (gets next assertion, builds it, repeats)
- `spekk builder --once` gets the next priority assertion, builds it, then exits
- Exit code 0 on success after single build
- Combines with `--spec` to build one assertion from a specific spec then exit
- Combines with `--confirm` for a single prompted build
