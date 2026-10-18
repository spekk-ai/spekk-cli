---
id: deploy-installs-matching-cli
parent: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
status: done
branch: feat/sandbox-spekk-binaries
---

# The Deploy Step Installs the spekk CLI of the Agent-Client's Release

## Success Criteria

- The deploy step that `spekk sandbox create`, `provision`, and `deploy` share downloads `spekk-linux-<arch>` from the same release and for the same machine CPU as `sandbox-linux-<arch>`, and copies both to the machine.
- On the machine, the CLI is installed at `/opt/spekk/cli/spekk`, mode 0755, owner `agent:agent`, with the same staged-then-rename install as the agent-client, and `/usr/local/bin/spekk` links to it. A root-owned file at that path is replaced.
- When `~agent/.local/bin/spekk` is there, it becomes a link to `/opt/spekk/cli/spekk`. When it is not there, nothing is made.
- The deploy prints the release of both binaries. `spekk sandbox deploy` ends with `Agent and spekk CLI <version> deployed to "<name>".`
- Live check: in a container with SSH, after `create --provider none`, `/opt/spekk/cli/spekk version` prints the release, and `stat` shows `agent agent`.

**Tests:** `internal/sandbox/deploy_replace_test.go` (`TestInstallCommandInstallsBothIntoMissingDirectories`, `TestInstallCommandReplacesRunningExecutable`), `internal/sandbox/release_test.go` (`TestDownloadBinariesSelectsArchAssets`). Live check (2026-10-18): two `ubuntu:24.04` containers with sshd and `--provider none`, one with a root login and one with `ubuntu` and sudo. Before the create, the first one had the root-owned v1.32.0 of the issue history at `/opt/spekk/cli/spekk` and `~agent/.local/bin/spekk`. After `create`, `stat` shows `agent:agent 755 /opt/spekk/cli/spekk`, both paths link to it, `spekk version` as agent prints `v1.38.1`, and the agent-client prints `version: v1.38.1`.
