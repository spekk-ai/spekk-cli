---
id: prepare-machine-script
parent: sandbox-provider-abstraction
created: 2026-09-12T00:00:00Z
priority: 2
status: done
branch: sandbox-agent-arm64
depends-on: register-an-existing-machine
---

# spekk Publishes the Setup an Existing Machine Needs

spekk does not provision a machine it did not create — the operator prepares it.
`scripts/prepare-machine.sh` is the published artifact that does that
preparation, so an operator no longer has to reverse-engineer the contract from
`cloud-init.yaml`. It is the `--provider none` counterpart to cloud-init: the
same setup a droplet gets, minus the droplet-only hardening.

## Success Criteria

- `scripts/prepare-machine.sh` installs exactly what the agent needs: the
  `agent` user (home dir, passwordless sudo, `docker` + `systemd-journal`
  groups), Docker, Node.js + the Claude Code CLI, `git`/`gh`, and the spekk
  directories (`/opt/spekk`, `/etc/spekk`, `/var/log/spekk`).
- It ends by writing `/opt/spekk/.provisioned` — the marker `create` and
  `provision` check — so a machine it prepared satisfies
  `register-an-existing-machine`.
- It deliberately omits the droplet-only hardening in `cloud-init.yaml` (a full
  `apt upgrade`, a default-deny UFW policy, fail2ban), which on a machine the
  operator already uses could lock them out or disrupt other services.
- It is arch-aware (uses `dpkg --print-architecture` for the Docker and `gh`
  apt sources, so it works on amd64 and arm64, e.g. a Raspberry Pi), targets
  Debian/Ubuntu (`apt-get`), requires root, and is safe to re-run (idempotent).
- It fails fast with a clear message when not run as root or when `apt-get` is
  absent.
- `docs/cli-reference.md` documents it under "A machine you already have."

**Note:** as a provisioning shell script it has no automated test; correctness
is verified by running it and by review. It duplicates the setup steps in
`internal/sandbox/cloud-init.yaml` and must be kept in step with that file —
change one, change the other. Collapsing the two into a single artifact (which
would also delete `renderCloudInit`) is tracked as the remaining follow-up in
`register-an-existing-machine`.
