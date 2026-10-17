---
id: machine-stage-lines
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
branch: feature/sandbox-machine-mode
depends-on: machine-create-input
---

# `create --machine` Writes One JSON Line for Each Stage

The worker reads stdout line by line and sets the stage of the launch from each line. A line that is not a stage line is noise the worker must skip, so stdout carries stage lines only.

**Tests:** internal/sandbox/machine_test.go

## Success Criteria

- The stages, in order:
  - `{"stage": "creating"}` after the input is valid, before the first cloud call.
  - `{"stage": "provisioning", "droplet_id": <int>, "ip": "<ip>"}` when the droplet is active with a public IP, before the cloud-init wait.
  - `{"stage": "registering"}` after cloud-init writes `/opt/spekk/.provisioned`, before the registration call.
  - `{"stage": "online", "agent_id": "<id>"}` after the agent is registered, the box is equipped, the agent service runs, and the record is `active`.
- On any error: `{"stage": "failed", "at": "<stage>", "message": "<text>"}`, where `at` is the last stage that started, and the exit status is 1. On `online`, the exit status is 0.
- While the command waits (the release download, the droplet active wait, the cloud-init wait, and each SSH step that equips the box), it writes the last stage line again, unchanged, every 30 seconds. The worker sees a line at least every 30 seconds and sets the same stage again. The CLI stops the repeat before it writes the `online` or `failed` line, so the final line is always the last line.
- Each line is one JSON object and one `\n`. Nothing else goes to stdout, from the CLI or from a child process (`ssh`, `scp`, `ssh-keygen`).
- `message` and each stderr line hold no secret. Before the CLI writes `message` or its final error, it replaces each secret of the input that is in the text with `[redacted]`.
- The record and the SSH key are under `$HOME/.config/spekk/` (or `$XDG_CONFIG_HOME/spekk/`), as today. A failure after the droplet exists keeps the record, so `destroy --machine` finds the droplet. The CLI does not destroy the droplet and does not revoke the agent on a failure: the caller does both.
- The droplet has the tags of a terminal create (`spekk-sandbox`, `spekk-kind:<kind>`, `spekk-host:<host>`, and `spekk-expires:<date>` for a one-off sandbox), and one tag `<key>:<value>` for each `--label`. The `--label` tags are in the droplet create request, so the droplet has them from the start, also when the CLI stops before the droplet is active.

## Verification

- One test runs `create --machine` to `online` against a fake DigitalOcean API and a fake control host, with `ssh` and the cloud-init wait stubbed. It checks the four lines in order, that stdout has nothing else, the tags of the droplet, and the record.
- One test makes the cloud-init wait take some intervals of a short repeat time, and checks that the `provisioning` line comes again unchanged during the wait, and that no line comes after `online`.
- The same test makes the deploy step fail and checks the `failed` line at `registering`, that no secret is in stdout or stderr, and that the record keeps the droplet id.
