---
id: register-command
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-self-register
depends-on: create-registers-agent
---

# `spekk sandbox register <name>` Registers a Sandbox Later

**Tests:** internal/sandbox/register_test.go, cmd/spekk/sandbox_help_test.go

## Success Criteria

- `spekk sandbox register <name> [--organization <id>]` needs `SPEKK_API_TOKEN` and `SPEKK_HOST`, and a recorded sandbox.
- It registers a new agent with the same request that `create` sends.
- Over SSH, it replaces the `SPEKK_AGENT_TOKEN` line in `/etc/spekk/agent.env` with the new token, keeps every other line, keeps the file mode `600` and the owner `agent`, and restarts `spekk-agent`. The token goes to the machine base64-encoded, never as plain text in the command.
- If that SSH step fails, it revokes the new agent and says so.
- On success, it stores the new agent id. If the record already had an agent id, it then revokes that old agent, because the old token is no longer on the machine. A failed revoke of the old agent is a warning that names the old id.
- `spekk sandbox --help` lists `register`.

## Verification

- `go test ./internal/sandbox` drives `register` against an `httptest` control host with the SSH seam stubbed. It checks the script that reaches SSH, the stored id, the revoke of the old agent, and the revoke of the new agent when SSH fails.
