---
id: serve-stays-claude-only
parent: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 3
status: not_started
depends-on: harness-selection-config
branch: feature/configurable-harness
---

# `serve` refuses a non-claude harness instead of spawning the wrong binary

`spekk serve` depends on Claude Code's `stream-json` protocol and is out of
scope for harness abstraction. Under a non-claude harness it must fail with a
clear message rather than spawning a binary that cannot speak that protocol.

## Success Criteria

- When the resolved harness is `claude-code`, `serve` behaves exactly as today.
- When the resolved harness is any non-claude harness (e.g. opencode via flag or
  `SPEKK_HARNESS`), `serve` exits with a non-zero status and a message stating
  that `serve` supports only the claude-code harness.
- `serve` does not spawn the non-claude binary and does not fall back to
  spawning `claude` silently — it refuses explicitly.
- A test asserts the refusal path for a non-claude harness selection.
