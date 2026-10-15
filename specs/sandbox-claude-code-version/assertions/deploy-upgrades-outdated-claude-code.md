---
id: deploy-upgrades-outdated-claude-code
parent: sandbox-claude-code-version
created: 2026-10-15T21:30:00Z
priority: 1
status: not_started
branch: fix/claude-code-version
depends-on: sandbox-installs-native-claude-code
---

# A Deploy Leaves Claude Code at or Above the Minimum Before It Restarts the Agent

## Success Criteria

- The deploy step that `spekk sandbox deploy`, `create`, and `provision` share reads the version of `claude` that the `spekk-agent` service would run. It runs `claude --version` with `systemd-run`, with the same `User=` and `EnvironmentFile=` as the unit, so the check has the same `PATH` as the service: the systemd default, or the `PATH` that `agent.env` sets. A test fails when the unit and the check name a different user or environment file.
- At or above `MinVersion`, the deploy installs nothing, and prints `Claude Code <version> (minimum <MinVersion>).`
- Older than `MinVersion`, or not read, the deploy prints what it found and installs the `stable` release with the same install and link lines as the provisioning paths. Then it reads the version again.
- When the install fails, the deploy fails with the install output. When the version is still older than the minimum after the install, the deploy fails with an error that names the version that it found and the minimum. The agent is not restarted, so the agent that runs keeps its turns.
- When the version cannot be read after the install, the deploy prints a `WARNING:` line with the cause, and continues to the restart. The client also fails open on an unreadable version, so a new output form does not stop deploys or turns.
- The check and the install happen before the service restart, and they do not stop a `claude` process. The native installer writes a new version file and moves the link, and a process that already runs keeps its old file. A deploy stops a running turn only where it did before: at the restart.
- Tests, with the SSH seam: a current version installs nothing and restarts; an old version installs, reads again, and restarts; a version that stays old fails the deploy, and no command after the failure (the agent install or the restart) runs; a version that cannot be read after the install gives a warning and a restart.

**Tests:** `internal/sandbox/claude_code_test.go` (`TestDeployKeepsClaudeCodeAtTheMinimum`), `internal/sandbox/ssh_user_test.go` (`TestDeployStagesAndInstallsForEachLogin`). Live check (2026-10-15): the version line, the install line, and the link line ran as root in an `ubuntu:24.04` container with an `agent` user and an old `claude` in `/usr/bin`. The version read gave 2.1.197 before the install and 2.1.292 after it.
