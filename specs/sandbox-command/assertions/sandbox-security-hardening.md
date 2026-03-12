---
id: sandbox-security-hardening
parent: sandbox-command
created: 2026-03-12T22:30:00Z
priority: 1
status: done
depends-on: sandbox-command-routing
branch: feature/sandbox-command
---

# Sandbox Security Hardening

## Requirement

Sandboxes must be hardened before any credentials are deployed to them. The provisioning flow in cloud-init and the create workflow must ensure the box is locked down first, then credentials are injected.

## Success Criteria

### SSH hardening (cloud-init.yaml)
- `PermitRootLogin` is set to `prohibit-password` (key-only, no password root login)
- `PasswordAuthentication` is set to `no` for all users
- SSHD is restarted after config changes
- fail2ban is installed and enabled with default SSH jail active

### Credential protection (cloud-init.yaml + create.js)
- `/etc/spekk/` directory is created with permissions `700`, owned by `agent:agent`
- `/etc/spekk/agent.env` is written with permissions `600`, owned by `agent:agent`
- The systemd unit runs as `User=agent`, so only the agent process can read its own credentials
- Root can still read the file (as root), but no other users can
- The `NOPASSWD` sudo grant is removed from the agent user — agent does not need sudo after provisioning

### Provisioning order (create.js)
- Security hardening steps in cloud-init run before the `.provisioned` marker is written
- `injectCredentials` in create.js writes agent.env owned by `agent:agent` with mode `600` (not root-owned)
- `configureGitCredentials` writes `.git-credentials` with mode `600` under the agent user's home directory (already the case, but verify)

### Docker isolation
- Docker socket access is restricted to the `docker` group (default, but agent user should only be in the docker group if needed)
