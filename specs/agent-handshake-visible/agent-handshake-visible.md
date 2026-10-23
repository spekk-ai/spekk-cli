---
id: agent-handshake-visible
created: 2026-10-23T06:00:00Z
priority: 1
---

# An agent that cannot complete its handshake is visible

## Problem

The agent client on a sandbox dials the control host, and it dials again with a backoff when the dial fails. A control host that refuses the agent, for example with HTTP 403 for a token that is not valid, makes every dial fail. The agent logs one line for each failure and continues. The systemd unit stays `active`, because the process runs.

`spekk sandbox status` shows the VM state, the provision marker, the systemd state of the agent service, and the binary versions. None of these shows whether the agent ever completed a handshake. Thus an agent that the control host refuses looks the same as a healthy one. A sandbox can stay in this state for months, and only a person who reads the log finds it (issue #215).

## Solution

- The agent client keeps a small JSON record of its connection on the machine: the time of the last successful handshake, and the time, HTTP status code, and error of the last failed dial. It writes the record after each dial.
- `spekk sandbox status` reads the record over SSH and shows both values. When the last dial failed, or the agent never connected, the status line has a warning.

## What this does not do

- **No change to a token, an agent registration, or a specific sandbox.** This is the general check only. To repair a refused sandbox is a separate operation (`spekk sandbox register`).
- **No control-host change and no protocol change.** The record stays on the machine.
- **No fleet check and no alert.** `spekk sandbox list` does not SSH to each machine, and an alert is a separate feature (issue #210).
- **No new `--json` output for `spekk sandbox status`.** The command has no JSON output today. The record itself is JSON, at `/var/log/spekk/agent-connection.json`, for a script that needs it.
