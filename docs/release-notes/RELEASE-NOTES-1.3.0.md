# Spekk CLI v1.3.0 — Cloud Sandboxes

New `spekk sandbox` command for provisioning and managing DigitalOcean droplet-based agent sandboxes.

⚠️ Right now this is for my admin use only, but conceivably if you get the right credentials together, you can use this to create sandboxes for yourself. It only creates the VM and deploys the agent. Connecting to Slack is a separate process partially automated through Django. I'm working towards full end-user-trigger-able automation, but need to think through a few things before then.

## New Commands

- **`spekk sandbox create`** — Provision a new DO droplet with cloud-init, wait for SSH readiness, inject credentials, configure git/gh, and deploy the agent client. Flags: `--name` (required), `--region` (default: nyc1), `--size` (default: s-2vcpu-4gb).
- **`spekk sandbox list`** — Display all tracked sandboxes in a table format.
- **`spekk sandbox status`** — Show detailed status including live DO API and SSH connectivity checks.
- **`spekk sandbox ssh`** — Open an interactive SSH session to a sandbox, with flag passthrough.
- **`spekk sandbox destroy`** — Tear down a sandbox droplet via the DO API and clean up local metadata. Supports `--force` to skip confirmation.
- **`spekk sandbox deploy`** — Redeploy the agent-client script to an existing sandbox via SCP, upgrade dependencies, and restart the spekk-agent systemd service.

## Architecture

- **`src/sandbox/do-api.js`** — DigitalOcean API client (native fetch)
- **`src/sandbox/store.js`** — Local metadata store persisting sandbox records to `~/.spekk/sandboxes.json`
- **`src/sandbox/templates.js`** — Bundled template loader for `cloud-init.yaml`
- **`src/sandbox/create.js`** — Full create workflow: droplet creation, polling, SSH setup, credential injection, agent deployment
- **`src/sandbox/cli.js`** — Command router with subcommand dispatch and help output
