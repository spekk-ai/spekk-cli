---
id: sandbox-create-auto-token
parent: sandbox-command
created: 2026-03-13T00:00:00Z
priority: 1
status: done
depends-on: sandbox-token-generator
---

# Sandbox Create: Auto-Generate Agent Token

`spekk sandbox create` auto-generates a unique `SPEKK_AGENT_TOKEN` per sandbox using `generateAgentToken()`. The token is no longer required as an environment variable. The `registerAgent()` API call is removed — agents are registered manually in Django admin.

## Success Criteria

- `SPEKK_AGENT_TOKEN` is removed from `REQUIRED_ENV_VARS` in `create.js`
- `createSandbox()` calls `generateAgentToken()` to produce a fresh token at the start of each create run
- The generated token (not any env var) is written to `/etc/spekk/agent.env` as `SPEKK_AGENT_TOKEN` on the sandbox
- `registerAgent()` function and its call are removed from `create.js`
- `DO_API_TOKEN` is used only for DigitalOcean API calls; it is never injected into the sandbox env
- The final summary printed to stdout includes the generated token clearly labeled, e.g.:
  ```
  Sandbox created successfully:
    Name:           spekk-adarose
    IP:             1.2.3.4
    AGENT_TOKEN:    <generated-token>

  Next: Add this agent in Django admin at https://$SPEKK_HOST/staff/agent/agent/add/
    - Name: adarose
    - Sandbox ID: spekk-adarose
    - Auth token: <generated-token>
  ```
- `SPEKK_HOST` in the summary URL uses the bare hostname (no scheme prefix)
