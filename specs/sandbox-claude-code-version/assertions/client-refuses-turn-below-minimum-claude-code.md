---
id: client-refuses-turn-below-minimum-claude-code
parent: sandbox-claude-code-version
created: 2026-10-15T21:30:00Z
priority: 1
status: done
branch: fix/claude-code-version
depends-on: claude-code-minimum-version-defined-once
---

# The Sandbox Client Logs Its Claude Code Version and Refuses an Older One

## Success Criteria

- At start, the client logs the `claude` path that it found on its `PATH` and the version, for example `Claude Code 2.1.301 at /usr/local/bin/claude (minimum 2.1.292)`. When the version is older or cannot be read, the log line says so. The client starts in all cases, so that it can tell the control host why it refuses.
- Before each turn, the client runs `claude --version` again. A self-update or a deploy can change the version while the client runs, and the check costs about 20 ms.
- When the version is older than `MinVersion`, or `claude` is not found or its version cannot be read, the client does not start `claude` for the turn. It sends an `error` frame with `agent_session_id`, and an `error` text that names the version and the path that it found (or why there is none), the minimum, and the remedy: run `spekk sandbox deploy` for this sandbox to install a current version.
- The `error` text is fewer than 300 characters and is the whole message, because the control host keeps only `error`, and a place that shows it can cut it at 300 characters. The frame has no new field, so the protocol version does not change.
- Tests: a fake `claude` that reports an old version gets an `error` frame with the found version and the minimum, and the fake records that no turn ran; a fake `claude` that reports a current version runs the turn as before.

**Tests:** `cmd/sandbox/invoke_test.go` (`TestTurnRefusedOnOldClaudeCode`; every other client test runs a turn with a fake that reports 2.1.301). Live check (2026-10-15): the built client logged `Claude Code 2.1.301 at <home>/.local/bin/claude (minimum 2.1.292)` with the native binary first on `PATH`, a refusal warning that names `/usr/bin/claude` 2.1.197, and a not-installed warning with no `claude` on `PATH`.
