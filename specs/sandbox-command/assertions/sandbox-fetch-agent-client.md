---
id: sandbox-fetch-agent-client
parent: sandbox-command
created: 2026-03-12T23:00:00Z
priority: 1
status: not_started
depends-on: sandbox-create-workflow
branch: feature/sandbox-command
---

# Sandbox Fetches Agent Client from spekk-app Repo

## Requirement

The canonical agent client lives in `spekk-ai/spekk-app` at `infrastructure/droplet/agent-client.py`. The CLI must fetch it from GitHub at deploy time instead of bundling its own copy. This keeps the agent client in sync with the Django server it connects to.

## Success Criteria

### Fetch from GitHub
- `src/sandbox/templates.js` exports a `fetchAgentClient()` function that uses the GitHub Contents API (`GET /repos/spekk-ai/spekk-app/contents/infrastructure/droplet/agent-client.py`) with `GITHUB_TOKEN` for auth
- The fetched file is written to a temp location and returned as a path for SCP
- If the fetch fails (404, auth error), prints a clear error and exits

### Remove bundled agent client
- `src/sandbox/templates/agent-client.py` is deleted — the CLI no longer bundles its own agent client
- `readTemplate('agent-client.py')` is no longer called anywhere
- `getTemplatePath('agent-client.py')` is no longer called anywhere
- `cloud-init.yaml` remains bundled (it's CLI-specific provisioning config)

### Deploy flow updates
- `src/sandbox/create.js` `deployAgentClient` calls `fetchAgentClient()` to get the file, then SCPs it to the droplet
- `src/sandbox/deploy.js` does the same — fetches from GitHub, SCPs to droplet
- Both still install websockets into `/opt/spekk/.venv/` via uv after deploying

### Environment variable changes
- `SPEKK_HOST` is written to agent.env as a bare hostname (e.g. `spekk.ngrok.app`) — strip any `https://` or `http://` prefix and any trailing slash
- `WORKSPACE=/opt/spekk/workspace` is written to agent.env
- `SPEKK_SERVER_URL` is no longer written to agent.env — the agent client constructs its own WebSocket URL from `SPEKK_HOST` and `SPEKK_AGENT_TOKEN`
