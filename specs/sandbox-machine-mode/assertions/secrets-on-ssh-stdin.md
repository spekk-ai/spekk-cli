---
id: secrets-on-ssh-stdin
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
---

# No Secret Is in the Argument List of `ssh`, in Either Mode

Today `injectCredentials`, `configureGitCredentials`, and `Register` (through `buildTokenScript`) pass the script as the last argument of `ssh`. The script holds the env file or the token in base64, so each process on the host can read it in `/proc/<pid>/cmdline` while `ssh` runs. For a non-root login, `sudoWrap` puts the whole script in the argument list again, as base64.

**Tests:** internal/sandbox/credential_boundary_test.go

## Success Criteria

- Each script that holds a secret goes to the machine on the stdin of `ssh`. The remote command is `bash -s` for root, and `sudo bash -s` for another login user. These are the only words after the host in the argument list.
- The scripts that hold a secret: the env file write (`injectCredentials`), the git credential setup (`configureGitCredentials`), the agent token write (`Register`), and the machine-mode clone (`machine-git-access`).
- A script on stdin keeps the base64 encoding of each value in it, so no character of a secret can change the script.
- A failed step still reports the output of `ssh`, as today.
- The scripts with no secret (the probe, the version read, the deploy, the teardown) do not change.

**Note:** the base64 text of a secret is the secret. A test that looks only for the clear value passes on `main`.

## Verification

- One test in `credential_boundary_test.go` runs the credential injection, the git credential setup, and `Register`, for a root login and for a non-root login, with a stubbed `ssh`. It fails if an argument holds a secret in clear text, or a base64 run that decodes to text that holds a secret. It fails on `main`.
