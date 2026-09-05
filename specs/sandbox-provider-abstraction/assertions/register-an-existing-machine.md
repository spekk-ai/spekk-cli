---
id: register-an-existing-machine
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: done
depends-on: cli-provider-dispatch
branch: feat/manual-sandbox
---

# Spekk Registers a Machine It Did Not Create, and Does Not Provision It

Not every sandbox has to be a droplet. A machine the operator already has — bare metal, another cloud, a spare box under a desk — can run an agent just as well, and reaching it needs only an address and a key.

What spekk must not do is treat that machine as its own. The first design here generated a 140-line bash script that replicated cloud-init and ran it as root over SSH: it upgraded every package, rewrote the firewall to allow only port 22, and created a passwordless-sudo user. On a droplet spekk just made, that is housekeeping. On a machine somebody else runs, it is an outage — an operator whose sshd listens on 2222 is locked out of their own server, and whatever else that machine was serving is firewalled off.

It was also a second copy of the provisioning steps. The proof that copies drift is already in this repository: the `spekk-agent` unit is written both in `commands.go` and in `cloud-init.yaml`, and the two disagree about `WorkingDirectory` today. A third copy would have gone the same way.

So spekk does not provision a machine it did not create. The operator prepares it; spekk checks that it is ready, then does the part that is genuinely spekk's: inject credentials, deploy the agent, and start it.

## Success Criteria

- `spekk sandbox create --ip <address> --ssh-key <path>` registers an existing machine. `--provider none` says the same thing explicitly, and naming a machine with either flag infers it.
- Registration validates before it records: both flags are required, the private key must exist, and its path is stored absolute so later commands resolve it from any working directory. A failure here records nothing, because a typo should not cost the operator an entry they then have to force away.
- Create confirms `/opt/spekk/.provisioned` on the machine before it injects anything, and its error says both possible causes — the machine is not prepared, or the connection failed — because the check cannot tell them apart.
- After the check, the flow is exactly the flow a droplet gets: inject credentials, configure git credentials, deploy the agent binary, start the service.
- `spekk sandbox destroy` never destroys the machine. It stops and disables the agent, removes `/etc/spekk/agent.env`, the agent's `.git-credentials`, and its `gh` configuration, and then removes the local record.
- Removing those secrets does not depend on the service existing. A create that died before the agent was installed still injected them, and a stop that fails on a unit that was never there must not be the reason a GitHub token stays on somebody's server.
- A teardown that fails stops the command and keeps the local record, because deleting it would leave an agent running with live credentials and nothing pointing at it. `--force` is what clears the record of a machine that is gone for good.
- The local key pair is removed only when spekk generated it, so an operator-supplied `--ssh-key` is never deleted.

## Known gap

Spekk does not yet publish the setup an operator needs to run. The contract is `/opt/spekk/.provisioned` plus the packages and the `agent` user that `cloud-init.yaml` installs, and today that file is readable only as cloud-init YAML. The fix is to make one artifact serve both paths — a plain script that DigitalOcean accepts as user-data and an operator can run by hand — which also deletes `renderCloudInit` and the second copy of the systemd unit. That is follow-up work, not part of this assertion.
