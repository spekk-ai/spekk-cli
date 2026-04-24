---
id: sandbox-store-file-permissions
parent: golang-security-hardening
created: 2026-04-24T12:00:00Z
priority: 1
status: done
branch: feature/golang-migration
---

# Sandbox metadata file has restrictive permissions

The `sandboxes.json` file written by `internal/sandbox/store.go` uses `0o600` permissions (owner read/write only) instead of `0o644`, since it contains infrastructure metadata including droplet IPs and IDs.

## Success Criteria

- `os.WriteFile` call in `writeSandboxes` uses file mode `0o600`
- Existing tests continue to pass
