---
id: manual-provider
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: done
depends-on: provider-interface
branch: dev/headless-sandbox
---

# A manual provider registers any SSH-reachable machine as a sandbox

The user supplies an IP address and SSH key. Spekk provisions the machine over SSH (same packages and setup as cloud-init), then deploys the agent. No cloud API required.

## Success Criteria

- `spekk sandbox create --provider manual --name X --ip 1.2.3.4 --ssh-key ~/.ssh/id_ed25519` registers the machine as a sandbox
- Provisioning runs the cloud-init content as a shell script over SSH (installs Docker, Node, gh CLI, Claude Code, creates agent user, sets up directories, configures firewall)
- After provisioning, the standard deploy flow runs: credential injection, agent binary deployment, systemd service start
- `--ip` and `--ssh-key` are required when `--provider manual`; passing `--region` or `--size` is an error
- `Destroy` with a manual sandbox stops the agent service and removes local metadata but does NOT destroy the machine
- `Status` with a manual sandbox checks reachability and agent service status via SSH only — no cloud API calls
- The provisioning script is idempotent — running it on an already-provisioned machine does not break it

**Tests:** internal/sandbox/manual_provider_test.go
