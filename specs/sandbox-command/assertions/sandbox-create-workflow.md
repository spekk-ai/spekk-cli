---
id: sandbox-create-workflow
parent: sandbox-command
created: 2026-03-12T18:00:00Z
priority: 1
status: done
branch: feature/sandbox-command
depends-on: do-api-client
---

# Sandbox Create Workflow

## Requirement

`spekk sandbox create --name <name>` provisions a full sandbox: creates a DO droplet, waits for it to be ready, injects credentials, deploys the agent client, registers the agent in Django, and saves metadata locally.

## Success Criteria

- `src/sandbox/create.js` exports an async `createSandbox({ name, region, size })` function
- The function calls `listSSHKeys()` from the DO API client and uses the first available key (or errors if none found)
- It fetches `cloud-init.yaml` from the spekk-app GitHub release and passes it as `user_data` to `createDroplet`
- The droplet is created with the tag `spekk-sandbox` and the name `spekk-{name}`
- After droplet creation, polls `getDroplet(id)` every 5 seconds until status is `active` and a public IPv4 address is available (timeout after 5 minutes)
- After the droplet is active, polls SSH connectivity (TCP connect to port 22) until it succeeds, then checks for the `/opt/spekk/.provisioned` marker file via SSH (timeout after 10 minutes)
- Reads credentials from environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION`, `GITHUB_TOKEN`, `SPEKK_AGENT_TOKEN`, `SPEKK_HOST` -- if any are missing, prints which are missing and exits with an error
- AWS Bedrock is the default model provider for agent sandboxes (not direct Anthropic API)
- SSHes into the droplet as `root` and writes `/etc/spekk/agent.env` with all six credential values, sets permissions to 600
- The agent.env file also includes `CLAUDE_CODE_USE_BEDROCK=1` so Claude Code on the droplet uses Bedrock by default
- Configures git credentials and gh CLI for the `agent` user via SSH commands (same steps as `setup-credentials.sh`)
- Deploys the Go agent binary and starts the `spekk-agent` systemd service (via `deployAgent`)
- Calls the Spekk API (`POST /api/agents/`) with the agent name, droplet IP, and agent token to register the sandbox
- Saves metadata to the local store: `{ dropletId, ip, region, size, createdAt, status: 'active' }`
- Prints a summary: sandbox name, IP, region, size, and "Agent connected" or connection error
- If any step fails after droplet creation, prints the error and the droplet IP so the user can debug manually (does not auto-destroy)
