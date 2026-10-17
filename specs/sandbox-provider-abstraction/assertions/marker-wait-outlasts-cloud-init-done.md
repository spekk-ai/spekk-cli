---
id: marker-wait-outlasts-cloud-init-done
parent: sandbox-provider-abstraction
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
depends-on: create-resumes-after-provisioning-timeout
branch: fix/provisioned-marker-race
---

# The Provisioning Wait Does Not Fail on a Marker That Arrives Just After Cloud-Init Is Done

`spekk sandbox create` stopped with "cloud-init finished without writing /opt/spekk/.provisioned" and revoked its agent, although provisioning had succeeded (issue #282). On the machine after the failure, the marker was there, with the same second as the "finished" line of cloud-init, and `cloud-init status --long` reported `status: done` and `errors: []`.

The cause: one probe tested the marker first and then asked `cloud-init status`. When the last `runcmd` step wrote the marker between the two, the probe read "no marker" and "done" together. The wait read that as "the marker will not come" and stopped on the first such probe.

## Success Criteria

- One probe asks cloud-init for its status before it tests the marker, so a `done` status cannot come from a later moment than the marker test.
- When cloud-init reports `done` and the marker is not there, the wait continues to poll for the marker for a grace period of 30 seconds. If the marker appears in that period, the wait succeeds. The grace period continues past the `--provision-timeout` deadline when cloud-init reports `done` just before it.
- The wait stops early only on a real cloud-init failure: `cloud-init status --long` reports `status: error`, or its `errors` value is not empty. The error message has the output of `cloud-init status --long` and the last line of `/var/log/cloud-init-output.log`.
- When the grace period ends with no marker, the error says that cloud-init finished without the marker, and it has the same status output and the last log line.
- No agent is revoked while the wait is in its grace period: the revoke runs only after the wait returns an error.
- Machine mode (`create --machine`) writes the same stage lines as before: `creating`, `provisioning` (written again by the keepalive during the grace period), `registering`, `online`.

## Verification

- A test drives `CreateMachine` through the real wait, with a fake probe runner where the marker appears 2 seconds after cloud-init reports `done`. The create ends `online` with the stage lines above. This test fails on the code before this assertion.
- The existing early-stop test covers `status: error` and a non-empty `errors` value, and checks that the message has the status output.
