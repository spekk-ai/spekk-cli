---
id: sandbox-agent-env-loading
parent: sandbox-command
created: 2026-03-12T22:00:00Z
priority: 1
status: not_started
depends-on: sandbox-create-workflow
branch: feature/sandbox-command
---

# Sandbox Agent Environment Loading

## Requirement

The agent systemd service must load credentials from `/etc/spekk/agent.env` and construct the correct WebSocket URL for the Spekk server. Currently the systemd unit has no `EnvironmentFile` directive, so credentials are never loaded. Additionally, the agent client needs `SPEKK_SERVER_URL` set to the full WebSocket endpoint including the agent's auth token.

## Success Criteria

- `cloud-init.yaml` systemd unit includes `EnvironmentFile=/etc/spekk/agent.env` so all credentials are available to the agent process
- `src/sandbox/create.js` `injectCredentials` writes `SPEKK_SERVER_URL=wss://{host}/ws/agent/{token}/` to `/etc/spekk/agent.env` (constructed from `SPEKK_HOST` and `SPEKK_AGENT_TOKEN`, stripping any `https://` prefix from the host and replacing with `wss://`)
- `src/sandbox/create.js` `injectCredentials` also writes `SPEKK_AGENT_NAME=spekk-{name}` to the env file so the agent identifies itself correctly
- The agent-client.py `SERVER_URL` default remains `ws://localhost:8080` but is overridden at runtime by the `SPEKK_SERVER_URL` env var loaded from agent.env
- After `systemctl start spekk-agent`, the agent process has access to all env vars from agent.env (AWS creds, GitHub token, Spekk server URL)
