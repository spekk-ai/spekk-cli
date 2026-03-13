---
id: sandbox-bundle-agent-client
parent: sandbox-command
created: 2026-03-13T01:00:00Z
priority: 1
status: in_progress
locked-by: builder-Williams-MBP.local-82655-1773431907
---

# Bundle agent-client.py with CLI

`agent-client.py` is bundled in `src/sandbox/templates/` so that `create` and `deploy` work without a GitHub API call. `fetchAgentClient()` is replaced with a local read.

## Context

`fetchAgentClient()` currently fetches `agent-client.py` from the GitHub API at `spekk-ai/spekk-app`. This fails when the `GITHUB_TOKEN` in the environment doesn't have access to that org. Since `agent-client.py` is part of the CLI's provisioning workflow, it should be bundled alongside `cloud-init.yaml`.

## Success Criteria

- `agent-client.py` from `infrastructure/droplet/agent-client.py` (spekk-app repo) is copied into `src/sandbox/templates/agent-client.py` in spekk-cli
- `fetchAgentClient()` in `templates.js` is replaced with a synchronous `getTemplatePath('agent-client.py')` call (no network request, no temp file needed — just return the path directly)
- `create.js` and `deploy.js` work without any GitHub API call for the agent client
- `agent-client.py` is listed in the `files` array in `package.json` so it's included in npm publishes
- The old `fetchAgentClient` export is removed from `templates.js`
