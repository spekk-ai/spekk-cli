---
id: manual-provider
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: draft
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

## Why this is still draft

The implementation below works, but two questions are open, and both change what the code should be.

First, whether "manual" is a provider at all. A machine spekk did not create has no lifecycle to own: `Destroy` cannot destroy it, and `Status` has no API to ask. Three methods describing an absence is a sign the type is wrong. It may belong as a provisioning mode on `create`, with `provider: none` in metadata, leaving `Provider` to mean "the cloud that owns this machine". A manual sandbox stays a full member of the registry either way.

Second, how much of the operator's machine spekk may change. The generated script runs `apt-get upgrade -y` on a host that may be in production, and `ufw --force enable` with only `22/tcp` allowed — which locks out an operator whose sshd listens elsewhere, and firewalls off whatever else the machine was already serving. A register-only design, where the operator prepares the machine and spekk only injects credentials and deploys the agent, removes the whole class. Running the same cloud-init the DigitalOcean path uses is the other option, and it keeps one definition of a provisioned sandbox instead of two that drift.

Teardown does not wait on those questions, because it is the same either way. Destroying a manual sandbox stops the agent, removes `/etc/spekk/agent.env` and the git credentials, and verifies the unit is no longer active. If any of that fails the command stops and keeps the local record: a machine that survives destroy keeps whatever is left on it, so a silent success would strand live AWS keys and a GitHub token with nothing pointing at them. `--force` still clears the record for a machine that is gone for good. The local key pair is removed only when spekk generated it, so an operator-supplied `--ssh-key` is never deleted.

One known defect stands regardless: the script is passed as an SSH argument, so the remote login shell runs it and the `#!/bin/bash` line is inert. On a host where root's shell is dash, provisioning fails on `set -o pipefail`. Piping to `bash -s` on stdin fixes it.
