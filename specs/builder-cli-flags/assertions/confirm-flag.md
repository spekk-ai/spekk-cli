---
id: confirm-flag
parent: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
status: done
---

# --confirm Flag Asks Before Building

## Description

The `--confirm` flag adds interactive confirmation before each build, allowing supervised autonomous mode.

## Success Criteria

- `spekk builder --confirm` shows assertion details and prompts `Build this? [y/n]`
- `y` or `Enter` proceeds with build
- `n` skips to next assertion
- `q` exits the builder
- Works with `--all` for supervised continuous mode
- Works with `--spec` to confirm only within that spec
