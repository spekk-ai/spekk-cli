---
id: sandbox-go-deploy
parent: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-go-release
depends-on: sandbox-release-downloader
---

# Sandbox Create and Deploy Share One Go Deploy Path

`sandbox create`, `sandbox provision`, and `sandbox deploy` install the Go agent binary through one shared function. Deployment can replace a running agent for root and non-root SSH users. This fixes the remaining deployment defect in issue #214.

## Success Criteria

- `deployAgent` is the shared installation path for create, provision, and deploy. It uses the recorded SSH login user.
- Every upload stages the binary in the login user's home directory. SCP never writes to the running executable or a fixed name in a shared temporary directory.
- Installation uses the absolute upload path from the login user's home, even when privilege escalation changes the working directory or `HOME`.
- Installation prepares an executable file on the destination filesystem, then replaces `/opt/spekk/agent-client` by rename. A running process retains its old executable until the service restart.
- Upload or preparation failure leaves the installed binary intact and prevents the service restart. Temporary destination files are removed after success or failure.
- The installation script creates `/opt/spekk/workspace` and `/var/log/spekk`, sets their required ownership, and writes `/etc/systemd/system/spekk-agent.service`.
- The systemd unit runs `/opt/spekk/agent-client` and appends stdout and stderr to `/var/log/spekk/agent.log`.
- The service reload, enable, and restart commands run after the replacement succeeds.
- Root runs the installation script directly. A non-root login uses the shared privilege helper.
- Create and deploy download the versioned release binary. Create uses the embedded cloud-init template for provisioning.
- The sandbox source has no Python, uv, pip, or venv deployment steps.

## Verification

`internal/sandbox/deploy_replace_test.go` runs the generated installation command against a copied Go executable. The child answers probes before and after replacement. The test checks failure handling, temporary-file cleanup, and a privilege wrapper that changes the working directory and `HOME`. `internal/sandbox/ssh_user_test.go` checks the production upload and SSH commands for root and non-root users, including upload failure.
