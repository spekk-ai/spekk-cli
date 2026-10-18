---
id: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
---

# A Sandbox's spekk Binaries Are Installed, Checked, and Reported

## Problem

Issue #210. A sandbox runs two spekk binaries from a release: the agent-client (`/opt/spekk/agent-client`) and the spekk CLI (`/opt/spekk/cli/spekk`, linked from `/usr/local/bin/spekk`). Only the agent-client has an install path that an operator controls. The cloud-init template and `scripts/prepare-machine.sh` install the CLI with `install.sh` from `main`, which gives the latest release at boot time, with no check of the file. That release is not always the release of the agent-client, and a sandbox made from a template image has the CLI of the day the image was built. `spekk sandbox deploy` does not touch the CLI, so the CLI on a sandbox stays at the version of the day it was made. A fleet audit found every sandbox behind. Nothing on the operator side shows which version a sandbox runs, so nothing shows drift, and nothing shows an agent-client that reports `dev` because a person copied a local build to it.

## Design

- **One install path for both binaries.** The deploy step that `create`, `provision`, and `deploy` share installs the agent-client and the CLI of the same release, for the CPU of the machine. The CLI goes to `/opt/spekk/cli/spekk`, owned by `agent`, and `/usr/local/bin/spekk` links to it. A `~agent/.local/bin/spekk` that is there (the root-owned copy of the issue history) becomes a link to the same file, so a schedule that calls it gets the new version. The cloud-init and `prepare-machine.sh` installs stay, because a template image build and a `--setup` script can use the CLI before the deploy step. The deploy step replaces that copy.
- **No auto-update.** Nothing replaces a binary on a sandbox on its own. The update is `spekk sandbox deploy <name>`, which already exists and already restarts the agent.
- **The release.** `--release <tag>` pins it. Without it, a CLI with a stable release version (`vX.Y.Z`) installs its own release, so the operator and the sandbox match, the same rule that machine mode uses. Any other build (`dev`, a commit) installs the latest release, and says so. A sandbox never gets an unstamped binary from the operator.
- **sha256.** GitHub publishes a `sha256:` digest for each release asset (`GET /repos/spekk-ai/spekk-cli/releases/tags/<tag>`, field `digest`). The operator reads the digests once for each deploy, before it downloads, and the sandbox checks the uploaded file with `sha256sum -c` before it replaces anything. One check, on the machine, covers a bad download and a bad copy. An asset with no digest, or a file that does not match, stops the deploy, and the installed binaries stay as they are.
- **Report.** `spekk sandbox status <name>` already connects to the machine. It prints the version of the CLI and of the agent-client that it reads there. A version that is not the release of the operator's CLI (the latest release, for a build that is not a release), `dev`, or no version, gets a `WARNING:` on its line, with the command that fixes it.

## Rejected

- Auto-update, a timer, or an update by the agent itself. A sandbox runs real work, and a binary must not change under a running turn.
- A version column in `spekk sandbox list`. `list` reads only local records, and an SSH connection to each machine makes it slow and makes it fail when one machine is down.
- A new field that the agent reports to the control host. This change stays in the CLI.
- A `SHA256SUMS` asset in the release workflow. The releases on GitHub already have a digest for each asset, and a new asset helps only the next releases.
- A second sha256 check on the operator side. The check on the machine covers both the download and the copy.
