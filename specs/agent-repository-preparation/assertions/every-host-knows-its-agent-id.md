---
id: every-host-knows-its-agent-id
parent: agent-repository-preparation
created: 2026-10-23T15:00:00Z
priority: 1
status: done
branch: fix/agent-id-in-env-file
depends-on: agent-chat-repository-access
---

# Each registered sandbox has its agent ID in the agent env file

**Tests:** internal/sandbox/auth_test.go, internal/sandbox/register_test.go

The agent client reads `SPEKK_AGENT_ID` from `/etc/spekk/agent.env` to ask for repository access and for a Git token. Before this change, only machine mode wrote that line. A sandbox that `create`, `provision`, or `register` equipped had no agent ID, so each repository chat turn failed with "Repository preparation failed: agent or session ID is missing".

## Success Criteria

- The env file that `create` and `provision` write has `SPEKK_AGENT_ID=<agent id>` from the record, in the same place as the machine mode file. `provision` of a sandbox with no agent writes the ID of the agent that it registers.
- `register` replaces the `SPEKK_AGENT_ID` line together with the `SPEKK_AGENT_TOKEN` line, so the file never names an old agent. Every other line stays, and a failed restart puts back the old file with the old ID and the old token.
- A sandbox that was equipped before this change gets the line from `spekk sandbox register <name>` or from one manual line in its env file and a restart of `spekk-agent`. This change does not edit a running machine.

## Verification

- The pinned env file test of `buildEnvContent` includes the `SPEKK_AGENT_ID` line.
- The token script test runs the script in a local shell and checks the new ID and the new token after success, and the old ID and the old token after a failed restart.
