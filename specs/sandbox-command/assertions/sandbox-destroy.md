---
id: sandbox-destroy
parent: sandbox-command
created: 2026-03-12T18:00:00Z
priority: 2
status: done
branch: feature/sandbox-command
depends-on: sandbox-metadata-store
---

# Sandbox Destroy

## Requirement

`spekk sandbox destroy <name>` deletes the DO droplet and removes local metadata.

## Success Criteria

- `spekk sandbox destroy <name>` looks up the sandbox in the local store to get the droplet ID
- Prints a confirmation prompt: "Destroy sandbox '<name>' (droplet <id>)? [y/N]" and waits for input
- If confirmed, calls `deleteDroplet(id)` from the DO API client
- On successful deletion, removes the entry from `~/.spekk/sandboxes.json` via `removeSandbox(name)`
- Prints "Sandbox '<name>' destroyed." on success
- If the DO API returns 404 (droplet already gone), still removes local metadata and prints a warning that the droplet was already deleted
- If the sandbox name is not found in the local store, prints an error and exits with code 1
- Supports `--force` or `-f` flag to skip the confirmation prompt
