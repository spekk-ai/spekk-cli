---
id: sandbox-command
created: 2026-03-12T15:00:00Z
priority: 1
---

# Sandbox Command

## Overview

The `spekk sandbox` command manages DigitalOcean droplet-based agent sandboxes. Each sandbox is an isolated Ubuntu 24.04 droplet running the Spekk agent client, which connects outward to the Spekk Django app via WebSocket to receive messages and invoke Claude Code.

This is a developer tool for provisioning, deploying, and managing agent infrastructure from the CLI.

## Commands

```
spekk sandbox create --name <name> [--region nyc1] [--size s-2vcpu-4gb]
spekk sandbox list
spekk sandbox status <name>
spekk sandbox ssh <name>
spekk sandbox destroy <name>
spekk sandbox deploy <name>
```

## Architecture

- Uses DigitalOcean API directly via Node.js `fetch` (no `doctl` dependency)
- Requires `DO_API_TOKEN` environment variable for API access
- Stores sandbox metadata locally in `~/.spekk/sandboxes.json`
- Cloud-init template and agent-client.py are bundled with the CLI package
- SSH key must be pre-registered in DigitalOcean; CLI uses the user's default SSH key

## Provisioning Flow (create)

1. Call DO API to create a droplet with cloud-init.yaml as user data
2. Poll droplet status until active, then poll SSH until cloud-init completes (`/opt/spekk/.provisioned` marker)
3. Read credentials from environment or prompt interactively: `ANTHROPIC_API_KEY`, `GITHUB_TOKEN`, `SPEKK_AGENT_TOKEN`, `SPEKK_HOST`
4. SSH in and run credential setup (write `/etc/spekk/agent.env`, configure git/gh for agent user)
5. Deploy `agent-client.py` to `/opt/spekk/`, install Python dependencies, start systemd service
6. Register the agent in Django via API (creates Agent model)
7. Print connection status and sandbox summary

## Reference Infrastructure

The provisioning templates live in the `spekk-agent-sandboxes` repo at `infrastructure/droplet/`:
- `cloud-init.yaml` -- droplet provisioning (Docker, Node.js, git, gh, Claude Code CLI)
- `setup-credentials.sh` -- post-provision secret injection
- `agent-client.py` -- WebSocket client that connects to Django
- `deploy-agent.sh` -- copies agent client and starts systemd service
