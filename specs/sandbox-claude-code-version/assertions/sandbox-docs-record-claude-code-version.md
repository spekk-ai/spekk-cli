---
id: sandbox-docs-record-claude-code-version
parent: sandbox-claude-code-version
created: 2026-10-15T21:30:00Z
priority: 2
status: not_started
branch: fix/claude-code-version
depends-on: deploy-upgrades-outdated-claude-code
---

# The Sandbox Docs Record the Claude Code Minimum

## Success Criteria

- `docs/advanced/sandbox-architecture.md` says that the agent runs the native Claude Code from `/usr/local/bin/claude`, that the minimum lives in `internal/claudecode`, and that the client refuses a turn on an older version with an `error` frame.
- `docs/cli-reference.md` says, in `spekk sandbox deploy`, that a deploy upgrades Claude Code to the `stable` release when it is older than the minimum, before the restart. The `create` text and the `prepare-machine.sh` text say that the native installer installs Claude Code as `agent`.
- The docs name the constant, not its value, so a change of the minimum changes one file.
