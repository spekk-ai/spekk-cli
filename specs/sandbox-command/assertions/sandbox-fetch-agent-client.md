---
id: sandbox-fetch-agent-client
parent: sandbox-command
created: 2026-03-12T23:00:00Z
priority: 1
status: not_started
depends-on: sandbox-create-workflow
branch: feature/sandbox-command
---

# Agent Client Is Fetched from spekk-app, Not Bundled

## Requirement

The canonical agent client lives in `spekk-ai/spekk-app` at `infrastructure/droplet/agent-client.py`. The CLI fetches it from GitHub at deploy time. No agent client code exists in this repo.

## End State

After this assertion is true:

- `src/sandbox/templates/agent-client.py` **does not exist** — deleted from this repo
- `src/sandbox/templates/` contains only `cloud-init.yaml` (CLI-specific provisioning config)
- The CLI fetches agent-client.py from GitHub on every `create` and `deploy`
- `agent.env` on the droplet contains `SPEKK_HOST` as a bare hostname (no scheme prefix), `WORKSPACE`, and `SPEKK_AGENT_TOKEN` — but NOT `SPEKK_SERVER_URL` (the agent client builds its own WebSocket URL)
- `registerAgent` in `create.js` prepends `https://` to `SPEKK_HOST` when calling the Django REST API, since SPEKK_HOST is now a bare hostname

## Success Criteria

### What's removed
- `src/sandbox/templates/agent-client.py` is deleted
- No calls to `readTemplate('agent-client.py')` or `getTemplatePath('agent-client.py')` exist in the codebase
- No agent client Python code exists anywhere in this repo

### What's added
- `src/sandbox/templates.js` exports `fetchAgentClient()` — uses GitHub Contents API (`GET /repos/spekk-ai/spekk-app/contents/infrastructure/droplet/agent-client.py`) with `GITHUB_TOKEN` from env for auth
- The fetched file is decoded from base64 (GitHub API returns base64-encoded content), written to a temp file, and the path is returned for SCP
- If the fetch fails (404, auth error, network), prints a clear error with the HTTP status and exits

### What changes
- `src/sandbox/create.js` `deployAgentClient` calls `fetchAgentClient()` to get the file path, then SCPs it to the droplet
- `src/sandbox/deploy.js` does the same — fetches from GitHub, SCPs to droplet
- Both still install websockets into `/opt/spekk/.venv/` via uv after deploying
- `src/sandbox/create.js` `injectCredentials` writes `SPEKK_HOST` as a bare hostname (strips `https://`, `http://`, trailing slashes from the env var value)
- `src/sandbox/create.js` `injectCredentials` writes `WORKSPACE=/opt/spekk/workspace` to agent.env
- `src/sandbox/create.js` `injectCredentials` no longer writes `SPEKK_SERVER_URL` — the agent client constructs its own WebSocket URL
- `src/sandbox/create.js` `registerAgent` prepends `https://` to `SPEKK_HOST` when building the API URL (since SPEKK_HOST is now bare)
