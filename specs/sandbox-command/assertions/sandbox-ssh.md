---
id: sandbox-ssh
parent: sandbox-command
created: 2026-03-12T18:00:00Z
priority: 2
status: not_started
depends-on: sandbox-metadata-store
---

# Sandbox SSH

## Requirement

`spekk sandbox ssh <name>` opens an interactive SSH session to the sandbox droplet.

## Success Criteria

- `spekk sandbox ssh <name>` looks up the sandbox IP from the local store
- Spawns `ssh root@<ip>` as an interactive child process using `child_process.spawn` with `stdio: 'inherit'`
- Exits with the same exit code as the SSH process
- If the sandbox name is not found, prints an error and exits with code 1
- Passes through any additional flags after the name to the SSH command (e.g., `spekk sandbox ssh agent-1 -L 8080:localhost:8080`)
