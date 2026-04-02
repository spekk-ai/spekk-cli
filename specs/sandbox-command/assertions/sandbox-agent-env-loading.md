---
id: sandbox-agent-env-loading
parent: sandbox-command
created: 2026-03-12T22:00:00Z
priority: 1
status: done
depends-on: sandbox-create-workflow
branch: feature/sandbox-command
---

# Sandbox Agent Environment Loading

## Requirement

The agent systemd service loads credentials from `/etc/spekk/agent.env`, including `SPEKK_SERVER_URL` set to the full WebSocket endpoint.

## Success Criteria

- `cloud-init.yaml` systemd unit includes `EnvironmentFile=/etc/spekk/agent.env` so all credentials are available to the agent process
- `src/sandbox/create.js` `injectCredentials` writes `SPEKK_SERVER_URL=wss://{host}/ws/agent/{token}/` to `/etc/spekk/agent.env` (constructed from `SPEKK_HOST` and `SPEKK_AGENT_TOKEN`, stripping any `https://` prefix from the host and replacing with `wss://`)
- `src/sandbox/create.js` `injectCredentials` also writes `SPEKK_AGENT_NAME=spekk-{name}` to the env file so the agent identifies itself correctly
- After `systemctl start spekk-agent`, the agent process has access to all env vars from agent.env (AWS creds, GitHub token, Spekk server URL)
