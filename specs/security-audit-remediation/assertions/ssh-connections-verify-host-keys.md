---
id: ssh-connections-verify-host-keys
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 2
status: done
depends-on: sandbox-name-validated
branch: feature/spekk-sandbox-vulnrabilities
---

# SSH connections verify host keys after initial connection

Sandbox SSH commands do not blanket-disable host key verification. After the first connection to a sandbox instance, subsequent connections verify the host key to prevent MITM attacks. The `StrictHostKeyChecking=no` and `UserKnownHostsFile=/dev/null` flags are replaced with a per-sandbox known_hosts approach.

## Success Criteria

- Each sandbox gets a dedicated known_hosts file (e.g., in the spekk metadata directory)
- First SSH connection uses `StrictHostKeyChecking=accept-new` to auto-accept and record the key
- Subsequent SSH connections use `StrictHostKeyChecking=yes` with the sandbox-specific known_hosts file
- `UserKnownHostsFile=/dev/null` is no longer used — host keys are persisted
- When a sandbox is destroyed, its known_hosts file is cleaned up
- All SSH call sites in `internal/sandbox/commands.go` use the new approach
