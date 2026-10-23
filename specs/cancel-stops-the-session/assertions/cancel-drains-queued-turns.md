---
id: cancel-drains-queued-turns
parent: cancel-stops-the-session
created: 2026-10-23T05:00:00Z
priority: 1
status: done
---

# A cancel removes the session's queued messages

**Tests:** `cmd/sandbox/cancel_test.go`

## Success criteria

- When a `cancel` frame for a session arrives, the client signals the running process of that session (as before) and removes every message that is queued in that worker's `msgs` channel.
- No queued message that was in the queue at the time of the cancel starts a turn.
- The client sends one terminal `error` frame for each removed message. The frame has `agent_session_id` and an `error` text that says the turn was cancelled before it started. The send does not block the read loop: a terminal frame can wait up to `finalSendTimeout` for a connection.
- Frame order: the control host ends the oldest open turn of a session for each terminal frame (spekk-app `_resolve_turn`). Thus the client sends the frames for the removed messages only after the turn in flight sent its terminal frame, and in the order the messages were received. When no turn is in flight, the client sends them at once, in order. The session's runner sends all of these frames, so no frame of a later turn goes before them.
- A message that `Dispatch` accepts after the cancel runs as a usual turn.
- A session that has no cancel runs every queued message, in order, as before.

**Note:** the removal and the check of the in-flight turn (see `cancel-before-start-is-kept`) use the worker mutex, and the runner takes a message under the same mutex. Thus a message is either removed by the cancel or taken by the runner, never both and never neither.

## Test

Real tests through the real `AgentClient` dispatch path, with the fake `claude` of `reconnect_test.go`:

- Dispatch one turn, and queue two follow-ups while it runs. Send a `cancel` frame. Wait until the pool releases the worker. Assert that the fake `claude` ran exactly once, and that the host received three `error` frames for the session: first the frame of the signaled turn, then one "cancelled before it started" frame for each removed message.
- With no cancel, dispatch one turn and two follow-ups. Assert that all three turns run and that the host receives three `result` frames.
