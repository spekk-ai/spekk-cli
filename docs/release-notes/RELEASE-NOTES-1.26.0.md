# Spekk CLI 1.26.0 — A Sandbox Stops Being a Droplet

A sandbox could only be one thing: a DigitalOcean droplet that spekk created, billing Claude through the AWS Bedrock API. Three of those were assumptions rather than requirements. This release makes the machine, the cloud that owns it, and the account that pays for Claude each a separate choice.

Nothing changes for an existing sandbox. Bedrock is still the default, and a droplet created before this release keeps working with no action.

## The cloud is behind an interface

`Provider` now owns the machine lifecycle — create it, destroy it, report its state — and everything after the machine exists is provider-agnostic: waiting for provisioning, injecting credentials, deploying the agent, writing metadata. DigitalOcean is one implementation of it.

The interesting part is what the interface takes. An early version handed each provider an opaque instance handle, which meant `SandboxMeta` had to drop `dropletId` and `sshKeyId`. Every sandbox created by an earlier binary carries exactly those fields, so it loaded with an empty handle — and `destroy` skipped teardown when the handle was empty, removed the local record, and reported success. The droplet kept billing with nothing left on disk to identify it.

So a provider reads and writes named fields on the metadata it is given. The on-disk format is unchanged and purely additive, an absent `provider` reads as DigitalOcean, and no value has to mean two different things. Three guards came out of that failure: `destroy` always calls the provider and stops on its error, so a failed teardown is never followed by deleting the record that makes the machine findable; the DigitalOcean provider refuses outright when no droplet id is recorded; and a local key pair is deleted only when spekk generated it.

`spekk sandbox status` also stops failing when it cannot reach the API. It prints the stored fields, marks the status it is showing as `(stored)` rather than passing a stale value off as live, and shows the droplet id again.

## A sandbox can be a machine you already have

`spekk sandbox create --ip <address> --ssh-key <path>` registers a machine that already exists — bare metal, another cloud, a container on a box under a desk. `--provider` gains `none`, and naming a machine with either flag infers it.

A machine spekk creates is reached as root; one you already have may not admit root. `--ssh-user <user>` logs in as that user instead (an AWS Ubuntu AMI admits `ubuntu`), and spekk escalates the privileged steps — credential injection, agent deploy, teardown — with `sudo`, so the user needs passwordless sudo, as those AMIs grant by default.

**Spekk does not provision a machine it did not create.** The first design generated a shell script replicating cloud-init and ran it as root over SSH. On a droplet spekk just made, that is housekeeping. On somebody else's server it is an outage: it upgraded every package, and it rewrote the firewall to allow only port 22, which locks out an operator whose sshd listens elsewhere and cuts off whatever else that machine was serving. It was also a second copy of the provisioning steps, and copies drift.

The operator prepares the machine, spekk confirms `/opt/spekk/.provisioned`, and then does the part that is genuinely spekk's: inject credentials, deploy the agent, start it. `destroy` never destroys such a machine. It stops the agent, removes the credentials spekk put there, and drops the local record — and if any of that fails it keeps the record, because deleting it would leave an agent running with live credentials and nothing pointing at it.

## A sandbox can pay with a subscription

`spekk sandbox create --auth subscription` authenticates Claude with a token from `claude setup-token` instead of billing through Bedrock. `--auth bedrock` is the default and is what a sandbox gets when the flag is absent.

Only the chosen mode's credentials are required, so a subscription sandbox is not blocked by AWS keys it will never use, and only that mode's variables are written, so no sandbox carries both and leaves the choice to whichever Claude Code reads first.

`infrastructure/sandbox/setup-credentials.sh` moves a sandbox that already exists between modes. It rewrites the env file whole, which is what makes a switch a switch, and reads the file first so the switch is not also a wipe: everything survives except the variables the mode itself decides. It restarts the agent, because systemd reads `EnvironmentFile` only at start and a running agent would otherwise keep billing the old account with nothing to show that the switch had not taken.

## The agent binary reports its version

Every CLI target in the release build was stamped with the release version; the sandbox agent was not, so a released agent reported `version: dev` exactly like one built on a laptop. A deployed fleet could not be asked which release it was running. From this release forward it can — agents deployed before it still report `dev` until they are redeployed.

## Upgrading

**Take this release before you take a sandbox to a subscription.** The credential script in earlier releases wrote `ANTHROPIC_API_KEY` into `/home/agent/.bashrc.d/spekk.sh`, a live key in every agent login shell, outside the env file and outside what `destroy` removed. This release stops writing it, removes it when switching auth mode, and adds it to the paths teardown clears.

Three commands are stricter, each where being permissive lost something:

- `spekk sandbox create` refuses a name already recorded. Re-running it after a failed create was the obvious recovery, and it overwrote the record naming the machine that failure left running.
- `spekk sandbox destroy` refuses a DigitalOcean sandbox whose metadata names no droplet, unless `--force`. Removing that record would hide a machine that may still be billing.
- `spekk sandbox status` and `destroy` now report a provider they cannot build. `status` continues with the stored values; `destroy` stops, because it cannot tear down what it cannot reach.

A model pin does not survive a change of auth mode. `ANTHROPIC_MODEL` names a model for one API — a Bedrock sandbox pins an inference profile such as `us.anthropic.claude-sonnet-5`, which a subscription rejects outright — so a switch drops it, reports what it dropped, and writes a replacement only if you supply one for the mode you are moving to. Re-running in the mode a sandbox is already in is a rotation, not a switch, and keeps it.
