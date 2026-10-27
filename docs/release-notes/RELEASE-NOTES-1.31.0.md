# Spekk CLI 1.31.0 - A Sandbox on Google Cloud, and an Agent for arm64

A spekk sandbox could be a DigitalOcean droplet, an AWS instance from the CloudFormation template, or a machine you prepared yourself, and its agent ran only on x86-64. This release adds Google Cloud as a provider that spekk creates and destroys, and an agent build for arm64 machines such as a Raspberry Pi. It also stops a droplet from trusting every SSH key on the DigitalOcean account.

## A sandbox on Google Cloud

`spekk sandbox create --provider gcp --name <name>` makes a Compute Engine VM from Ubuntu 24.04 with the same cloud-init a droplet gets, deploys the agent, and records the VM. `spekk sandbox destroy` deletes the VM and its boot disk. The provider runs the `gcloud` CLI, so it uses the account, the project, and the zone that `gcloud` is configured with, and spekk keeps no Google Cloud credential of its own.

The existing flags keep their names: `--region` takes the zone, `--size` the machine type, `--project` the project ID, and `--vpc` the network. Spekk records the project and the zone it used, so `destroy` finds the VM after your `gcloud` configuration changes.

The VM is locked down by default:

- It has no service account. The agent runs code from the repositories it clones, and a token from the metadata server would give that code access to the project.
- It blocks the project's SSH keys. The guest agent gives each of those users sudo, and with sudo a user can read the credentials spekk injects. Log in with `spekk sandbox ssh`.
- A name that is already taken by a VM spekk did not make is never recorded, so `destroy` cannot delete somebody else's VM.

The image does not admit root over SSH, so spekk logs in as a `spekk` user with passwordless sudo, and the provisioning wait now logs in as the user the provider recorded. The new page [A Sandbox on Google Cloud](../advanced/sandbox-gcp.md) walks the whole path.

## An agent for arm64

Each release now publishes the agent for `linux/arm64` as well as `linux/amd64`. Spekk runs `uname -m` on the machine after it can reach it, and deploys the build that matches. It refuses an architecture it has no build for, and does not guess. The detection reads only what the machine prints, so the warning that recent OpenSSH versions print for a server without a post-quantum key exchange no longer breaks it.

`--release <tag>` on `create`, `provision`, and `deploy` pulls the agent binary from a specific release, for example an `exp-*` prerelease. `spekk update --version <tag>` installs a specific CLI release, prereleases included. The agent binary now downloads from the public release CDN, so provisioning needs no `GITHUB_TOKEN` for the download. The sandbox still needs one for the agent's own work.

`scripts/prepare-machine.sh` prepares a Debian or Ubuntu machine for `--provider none`: the `agent` user, Docker, Claude Code, `git` and `gh`, the spekk CLI, and the spekk directories. It leaves out the firewall and package upgrade that a droplet gets, because those could lock you out of a machine you already use. On a machine you also use yourself, `SPEKK_SHARE_USER=<login>` gives your login user read and write access to the agent's workspace.

Every new sandbox now has the spekk CLI. It is installed in `/opt/spekk/cli`, which the `agent` user owns, so `spekk update` needs no sudo, and every user on the machine can run it through `/usr/local/bin/spekk`.

## A droplet trusts only its own key

`spekk sandbox create` added every SSH key on the DigitalOcean account to a new droplet. Each of those keys logged in as root, and root reads the GitHub token and the model credential in `/etc/spekk/agent.env`. In a team account, every teammate could read the credentials of every sandbox. A new droplet now trusts only the key spekk generated for it. Log in with `spekk sandbox ssh`.

## Upgrading

- A droplet created before this release keeps the account keys it was created with. To remove them, destroy the sandbox and create it again.
- A machine prepared with a copy of `prepare-machine.sh` from `main` or from an `exp-*` build, before this release, keeps the CLI in `~agent/.local/bin` until you run the script again. The released script moves it.
- To run the agent on an arm64 machine, no `--release` pin is needed from this release on.
