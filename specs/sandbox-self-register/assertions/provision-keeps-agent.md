---
id: provision-keeps-agent
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-self-register
depends-on: register-command
---

# `provision` Keeps the Agent That the Sandbox Already Has

`spekk sandbox provision` rewrites `/etc/spekk/agent.env`. Users run it to change the auth mode or to push new credentials. A new token in that file would disconnect a registered agent, and a new agent would lose its schedules and its history.

**Tests:** internal/sandbox/register_test.go

## Success Criteria

- When the record has an agent id, `provision` reads the current `SPEKK_AGENT_TOKEN` from `/etc/spekk/agent.env` over SSH and writes the same token back. It makes no call to the control host.
- If it cannot read a token, it stops before it changes the machine, and it tells the user to run `spekk sandbox register <name>`. A failed SSH connection and a file with no token give different errors, and the SSH error is in the message.
- When the record has no agent id and `SPEKK_API_TOKEN` is set, `provision` registers a new agent as `create` does, and revokes it if a later step fails. If the final save of the record fails after the machine has the new token, the error names the new agent id. `--organization <id>` selects the organization, as it does for `create`.
- When the record has no agent id and `SPEKK_API_TOKEN` is empty, `provision` keeps its behavior from before this spec, and also prints how to register later.

## Verification

- `go test ./internal/sandbox` covers the three cases with the token read and the equip step stubbed.
