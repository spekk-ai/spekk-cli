---
id: create-resumes-after-provisioning-timeout
parent: sandbox-provider-abstraction
created: 2026-09-05T00:00:00Z
priority: 1
status: done
depends-on: cli-provider-dispatch
branch: fix/create-resumes-after-provisioning-timeout
---

# A Create That Stops Waiting Leaves a Sandbox That Can Be Finished

`spekk sandbox create` waited a fixed ten minutes for cloud-init to write `/opt/spekk/.provisioned`. A droplet does not know about the clock: on a slow night an apt upgrade plus the nodesource repository took eighteen, and the machine finished provisioning after create had given up on it. What the operator was left with was a running droplet, a record at `provisioning`, a message that offered `destroy`, and three steps - inject the credentials, configure git for the agent, deploy the agent - that no command runs on their own. `deploy` runs only the last. The way out was to reproduce `buildEnvContent` and `buildGitCredentialScript` by hand over SSH.

Two things follow. The wait has to be the operator's to set, and it has to say what it sees while it waits, because a silent ten minutes and a hung one look the same. And a record at `provisioning` has to be resumable from where create stopped, through the same code create would have run, so that finishing by hand is never the answer.

## Success Criteria

- `spekk sandbox create --provision-timeout <duration>` sets how long create waits for the marker. No flag means 30 minutes. A value that is not a positive Go duration is refused before anything is created.
- While it waits, create prints a progress line at most once a minute: how long it has waited, and the last line of `/var/log/cloud-init-output.log` on the droplet. One SSH round trip per poll reports the marker, `cloud-init status`, and that line together.
- The wait stops early, with a message that says why and carries the last log line, when `cloud-init status` reports `error`, or reports `done` without the marker. The marker is the last `runcmd` step, so `done` without it means it is not coming.
- `spekk sandbox provision <name>` finishes a record whose status is `provisioning`. It refuses any other status without `--force`, and with `--force` it provisions the record anyway.
- `provision` validates the environment variables of the sandbox's auth mode with the same check create uses, and it fails before it touches the machine when one is missing. The record carries the auth mode create chose, so `provision` needs no flag to ask for the same credentials; `--auth` overrides it, and a record with no mode reads as bedrock.
- `provision` checks the marker over SSH, then runs the steps create runs after its wait - inject credentials, configure git credentials, deploy the agent - in that order, through one function that create also calls, so the two cannot drift. It then sets the status to `active` and prints the same "Next: Register this agent on the control host" message create prints, with the token.
- The message create prints when it stops waiting names `spekk sandbox provision <name>` as the way to finish, and no longer offers `destroy` first.
