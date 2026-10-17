---
id: image-hygiene-gate
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 1
status: done
depends-on: image-build
branch: feature/sandbox-template-images
---

# The Hygiene Check Fails the Build When the Builder Has a Credential

**Tests:** internal/sandbox/image_test.go

## Success Criteria

- Before the image is made, one SSH session on the builder runs a clean step and then a check. Both are in one session, because the clean step removes the authorized keys, so a new SSH connection cannot log in after it.
- The clean step removes what makes each sandbox its own machine, and the logs: the SSH host keys (`/etc/ssh/ssh_host_*`), the content of `/etc/machine-id` (the file stays, empty, so systemd makes a new ID at first boot), `/var/lib/dbus/machine-id`, the authorized keys of root and of each user in `/home`, the shell history of root and of each user, and `/opt/spekk/.provisioned`. It removes `~agent/.claude` and `~agent/.claude.json` when there is no Claude Code sign-in (`~agent/.claude/.credentials.json`), because the base cloud-init writes the settings there. On the machine itself it also runs `cloud-init clean --logs` and truncates the files in `/var/log`.
- The clean step does not remove credentials. The check fails the build, with one line for each finding, when any of these is on the builder:
  - `/etc/spekk/agent.env`
  - a git credential file (`.git-credentials`) or a GitHub CLI login (`.config/gh/hosts.yml`) of root or of a user in `/home`
  - `/home/agent/.bashrc.d/spekk.sh`
  - `~agent/.claude` or `~agent/.claude.json` (they are still there only when they hold a Claude Code sign-in)
  - a Docker login (`.docker/config.json` with an `auth` value) or an npm token (`.npmrc` with `_authToken`) of root or of a user
  - a line that sets a variable whose name has `TOKEN`, `SECRET`, `API_KEY`, `PASSWORD`, or `ACCESS_KEY` to a value, in `/etc/environment`, `/etc/profile.d/`, or the shell startup files of root or of a user
  - a shell history file, an SSH host key, a `/etc/machine-id` that is not empty, an authorized keys file, the marker, `/var/lib/cloud/instance`, or a cloud-init log in `/var/log`
- When the check fails, no image is made, and the builder is destroyed.
- Images are private to the account: a DigitalOcean snapshot and a GCP custom image are private by default, and the build does not share them.

## Verification

- A test with a fake provider plants `/etc/spekk/agent.env` in a temporary root, and runs the real hygiene script against that root with a fake SSH runner. The build fails and names the file, `CreateImage` is not called, and the builder is destroyed.
