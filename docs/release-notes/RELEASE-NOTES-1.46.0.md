# Spekk CLI 1.46.0 - Status Shows a Refused Agent, and a Cancel Stops the Session

Before this release, a sandbox agent that the control host refused looked healthy. The `spekk-agent` service stayed `active`, and only the agent log showed the failed dials. A `cancel` frame stopped only the running Claude process of a session, so the queued messages of that session then started new turns.

This release records the handshake of the agent and shows it in `spekk sandbox status`. A `cancel` now stops the session: its running process and its queued messages. The protocol version does not change.

## `spekk sandbox status` shows the handshake

The agent client writes a record of its connection to `/var/log/spekk/agent-connection.json` after each dial. The record has the time of the last successful handshake, and the time, HTTP status code, and error text of the last failed dial. The record does not contain the agent token. At start, the agent reads the record on the disk, so a restart keeps the last success.

`spekk sandbox status <name>` reads the record over SSH and prints two new lines after `Agent service`. When the last dial failed, or the agent has a failure and never connected, the failure line ends with a warning:

```
Agent last connected: never
Agent last handshake failure: 2026-10-23T05:10:00.123Z (HTTP 403)  WARNING: the agent is not connected to the control host
```

When the machine has no record, for example with an agent binary older than 1.46.0, the lines say `unknown (no record; the agent-client may be older than this check)`. When SSH fails, they say `unknown`.

A protocol reject (close code 4004) comes after a completed handshake, so the record shows that dial as a success. The version lines of `status` still warn about an agent binary that is not the current release.

## A cancel stops the session

A `cancel` frame now does these steps for its session:

- It sends SIGTERM to the running Claude process, as before.
- It removes every queued message of the session. For each removed message, the agent sends one `error` frame with `agent_session_id` and the text `Cancelled before the turn started.`, so the control host can close each turn that it opened. These frames go after the final frame of the running turn, in the order the messages arrived.
- A cancel that arrives before the process of the turn starts stops that turn. The turn ends with the same `error` frame and starts no process.

A message that arrives after the cancel runs as usual. A repository preparation or an attachment download that is in progress when the cancel arrives finishes first. Then the turn stops before its process starts.

## A large stream frame on a slow link

The write deadline of a `stream` frame was a flat 5 seconds. On an uplink slower than about 2 MB/s, a large tool result closed a healthy connection in the middle of a turn. The deadline is now 5 seconds plus the time to send the frame at 256 KiB/s. A small frame keeps the 5-second bound, and a frame of 10 MB gets less than one minute.

## Upgrading

The record and the cancel changes are in the agent binary. After the release is published, use `spekk sandbox deploy <name>` on each authorized sandbox to install the new CLI and agent. Confirm the installed versions with `spekk sandbox status <name>`, and read the two new handshake lines.
