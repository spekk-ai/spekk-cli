---
id: non-root-login-user
parent: sandbox-provider-abstraction
created: 2026-09-02T00:00:00Z
priority: 1
status: done
depends-on: register-an-existing-machine
branch: dev/headless-sandbox
---

# A Machine That Does Not Admit Root Is Still a Sandbox

Spekk reaches a droplet it created as root, because it made the machine and decided that. A machine the operator already has decides for itself, and the common answer is no. An AWS Ubuntu AMI gives you `ubuntu` with passwordless sudo and disables root over SSH outright, so registering one and then reaching for root fails at the first step, before any credential is written.

The login user is therefore part of what the operator names, and root is what it means when they name nothing. The steps that genuinely need root are the three that touch `/opt/spekk`, `/etc/spekk` and the systemd unit: inject credentials, deploy the agent, tear it down. Each escalates. Everything else runs as the login user, because it does not need more.

Two things make this a security question rather than a convenience. The binary that root installs is staged by an unprivileged user first, so where it is staged decides who can substitute it. And the login user is stored and then interpolated into an ssh argument on every later command, so a value ssh reads as an option is a command that runs again and again on the operator's own machine.

## Success Criteria

- `spekk sandbox create --ip <address> --ssh-key <path> --ssh-user <user>` logs in as that user. An absent `--ssh-user` means root, and a sandbox recorded before the field existed reads as root, so an existing fleet is untouched.
- `--ssh-user` names a machine that already exists, so it infers `--provider none` on its own, and it is refused with `--provider digitalocean`. The inference and the refusal read one list of existing-machine flags: two lists disagree, and the operator who forgets `--ip` is then told about a flag they never typed.
- The privileged steps - credential injection, git credentials, agent deploy, teardown - run as root through one escalation helper, so the rule is stated once rather than at each call site. A root login runs the script as it is.
- Teardown removes the same credentials for a non-root login as for root. A machine spekk cannot fully clear keeps its local record, because a machine with live credentials and nothing pointing at it is worse than an extra entry.
- A non-root deploy stages the agent binary in the login user's home directory, never at a fixed name under `/tmp`. Every local user can write `/tmp`, and what root moves into `/opt/spekk` is what systemd then runs beside an env file holding AWS keys and a GitHub token.
- The login user is validated as a POSIX login name before anything is recorded. A value that starts with `-` is read by ssh as an option, not as part of the destination, so `-oProxyCommand=...` would run a command locally on every later `status`, `ssh`, `deploy` and `destroy`. An invalid value costs the operator no metadata entry.
- `waitForProvisioning` stays root-only. It runs only for a machine spekk created, where root is the contract.
