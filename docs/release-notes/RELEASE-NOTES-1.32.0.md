# Spekk CLI 1.32.0 - Each Stream Frame Names Its Session

The sandbox agent runs up to five sessions at the same time. A `stream` frame carried only the output line, so a control host that received lines from two sessions at the same time could not tell which session each line came from.

## The session id on every stream frame

Each `stream` frame now carries the turn's `agent_session_id`, as the `result` and `error` frames already do:

```json
{"type": "stream", "data": "<one line of claude stream-json>", "agent_session_id": "<the turn's session id>"}
```

A control host can use the field to send each line to the observers of the correct turn, for example a chat window that follows one message. The field is additive. The protocol version does not change, and a control host that does not read the field operates as before.

## Upgrading

Run `spekk sandbox deploy` for each sandbox to install the new agent. The deploy restarts the agent service, and this stops a turn that is running.
