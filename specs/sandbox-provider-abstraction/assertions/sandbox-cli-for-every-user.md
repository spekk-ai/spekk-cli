---
id: sandbox-cli-for-every-user
parent: sandbox-provider-abstraction
created: 2026-10-06T00:00:00Z
priority: 2
status: done
depends-on: prepare-machine-script
branch: fix/sandbox-cli-for-every-user
---

# Every User on a Sandbox Can Run the spekk CLI

Cloud-init and `prepare-machine.sh` install the spekk CLI as the `agent` user, so that `spekk update` replaces it with no sudo, and link it into `/usr/local/bin`. The install went into `~agent/.local/bin`. Ubuntu creates a home directory with mode `0750`, so the link worked for `agent` and root, and every other user got "command not found". That is every non-root login spekk uses: `ubuntu` on an AWS instance, and `spekk` on a Google Cloud VM.

The CLI goes into `/opt/spekk/cli` instead. `agent` owns that directory, so `spekk update` still needs no sudo, and the directory is outside every home, so the link works for every user. The temptation rejected here is to open `~agent` to other users, because that home holds the agent's git credentials.

## Success Criteria

- `cloud-init.yaml` and `prepare-machine.sh` create `/opt/spekk/cli` owned by `agent` with mode `0755`, install the CLI there as `agent` through `SPEKK_INSTALL_DIR`, and link `/usr/local/bin/spekk` to it. The AWS template gets the same lines through `go generate`.
- A user other than `agent` and root runs `spekk` through the link.
- `agent` runs `spekk update` with no sudo, and other users can still run the CLI after the update.
- `prepare-machine.sh` removes a copy that an earlier version of the script installed in `~agent/.local/bin`, because that copy would shadow the new one on agent's own PATH and stop receiving updates.
- A Go test fails when the link in either file points into a home directory.

**Tests:** internal/sandbox/cli_install_test.go
