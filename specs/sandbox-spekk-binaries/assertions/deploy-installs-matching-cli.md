---
id: deploy-installs-matching-cli
parent: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
status: not_started
branch: feat/sandbox-spekk-binaries
---

# The Deploy Step Installs the spekk CLI of the Agent-Client's Release

## Success Criteria

- The deploy step that `spekk sandbox create`, `provision`, and `deploy` share downloads `spekk-linux-<arch>` from the same release and for the same machine CPU as `sandbox-linux-<arch>`, and copies both to the machine.
- On the machine, the CLI is installed at `/opt/spekk/cli/spekk`, mode 0755, owner `agent:agent`, with the same staged-then-rename install as the agent-client, and `/usr/local/bin/spekk` links to it. A root-owned file at that path is replaced.
- When `~agent/.local/bin/spekk` is there, it becomes a link to `/opt/spekk/cli/spekk`. When it is not there, nothing is made.
- The deploy prints the release of both binaries. `spekk sandbox deploy` ends with `Agent and spekk CLI <version> deployed to "<name>".`
- Live check: in a container with SSH, after `create --provider none`, `/opt/spekk/cli/spekk version` prints the release, and `stat` shows `agent agent`.
