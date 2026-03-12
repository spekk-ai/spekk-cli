---
id: sandbox-list-and-status
parent: sandbox-command
created: 2026-03-12T18:00:00Z
priority: 2
status: not_started
branch: feature/sandbox-command
depends-on: sandbox-metadata-store
---

# Sandbox List and Status

## Requirement

`spekk sandbox list` shows all known sandboxes. `spekk sandbox status <name>` shows detailed status for one sandbox, including live data from the DO API.

## Success Criteria

- `spekk sandbox list` reads `~/.spekk/sandboxes.json` and prints a table with columns: Name, IP, Region, Status, Created
- If no sandboxes exist, prints "No sandboxes found." and exits with code 0
- `spekk sandbox status <name>` looks up the sandbox in the local store, then fetches live data from `getDroplet(id)`
- Status output includes: sandbox name, droplet ID, IP address, region, size, droplet status (from DO API), and created timestamp
- Status output also attempts an SSH check for the `/opt/spekk/.provisioned` marker and shows "Provisioned: yes/no"
- Status output checks `systemctl is-active spekk-agent` via SSH and shows "Agent service: active/inactive/unknown"
- If the sandbox name is not found in the local store, prints an error and exits with code 1
- If the DO API call fails (e.g., droplet was deleted outside the CLI), prints a warning and shows only local metadata
