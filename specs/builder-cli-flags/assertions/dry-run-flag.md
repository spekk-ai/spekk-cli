---
id: dry-run-flag
parent: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
status: done
---

# --dry-run Flag Shows What Would Be Built

## Description

The `--dry-run` flag shows what assertion would be built without actually launching Claude. Useful for previewing the queue.

## Success Criteria

- `spekk builder --dry-run` displays the next assertion's details:
  - Assertion ID
  - Title
  - File path
  - Priority
  - Status
- Does not launch Claude Code
- Exits immediately after displaying
- Works with `--spec` to preview filtered queue
