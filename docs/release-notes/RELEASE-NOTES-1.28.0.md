# Spekk CLI 1.28.0 - A Slow cloud-init No Longer Costs You the Droplet

`spekk sandbox create` waited a fixed ten minutes for cloud-init to finish on a new droplet. On a slow night, an apt upgrade plus the nodesource repository took eighteen. When the wait gave up, the droplet kept running and finished provisioning on its own, but the record stayed at `provisioning`, the command told you to destroy it, and the three steps that turn a provisioned machine into a sandbox - inject the credentials, configure git for the agent, deploy the agent - were reachable by no command. The way to finish was to reproduce them by hand over SSH.

## `--provision-timeout`

`spekk sandbox create --provision-timeout 45m` sets how long to wait for cloud-init, as a Go duration. The default is now 30 minutes. While it waits, `create` prints a progress line at most once a minute with the time waited and the last line of `/var/log/cloud-init-output.log` on the droplet, so a long apt run reads as alive rather than hung. If `cloud-init status` reports `error`, or reports `done` without writing `/opt/spekk/.provisioned`, the wait stops at once and says so, because the marker is the last step cloud-init runs and it is not coming.

## `spekk sandbox provision <name>`

A record that `create` left at `provisioning` can now be finished. `spekk sandbox provision <name>` checks that the marker exists on the machine, then runs the same three steps `create` runs after its wait, in the same order, through the same function, so the two cannot drift apart. It sets the record to `active` and prints a new agent token with the same registration reminder `create` prints. It needs the same environment variables as `create` for the sandbox's auth mode, and it refuses to start, before it touches the machine, when one is missing.

It refuses a record whose status is not `provisioning`; `--force` provisions one anyway. The record now carries the auth mode `create` chose, so `provision` asks for that mode's credentials without being told; `--auth` overrides it. A record written before this release has no mode recorded and reads as `bedrock`, which is what it was created with.

The message `create` prints when the wait runs out now names `spekk sandbox provision <name>` as the way to finish, and offers `destroy` second.
