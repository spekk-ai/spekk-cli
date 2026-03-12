---
id: sandbox-deploy
parent: sandbox-command
created: 2026-03-12T18:00:00Z
priority: 2
status: done
branch: feature/sandbox-command
depends-on: sandbox-metadata-store
---

# Sandbox Deploy

## Requirement

`spekk sandbox deploy <name>` redeploys the agent client to an existing sandbox. Useful for updating the agent code without reprovisioning the entire droplet.

## Success Criteria

- `spekk sandbox deploy <name>` looks up the sandbox IP from the local store
- Copies the bundled `agent-client.py` to `/opt/spekk/agent-client.py` on the droplet via SCP
- Installs/upgrades the `websockets` Python package via SSH
- Restarts the `spekk-agent` systemd service via SSH (`systemctl restart spekk-agent`)
- Checks `systemctl is-active spekk-agent` after restart and prints the result
- Prints "Agent redeployed to '<name>'." on success
- If the sandbox name is not found in the local store, prints an error and exits with code 1
- If SSH or SCP fails, prints the error with the IP so the user can debug manually
