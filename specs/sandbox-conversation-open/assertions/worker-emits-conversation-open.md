---
id: worker-emits-conversation-open
parent: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
status: not_started
depends-on: conversation-open-frame
---

# Worker Provisions the Spool and Emits the Frame

When the worker runs a Claude session it gives that session a private spool
directory to drop conversation requests into, then drains the directory and
emits a `conversation_open` frame per request on the same WebSocket connection,
stamping the authoritative session id itself.

## Success Criteria

- For each invocation in `cmd/sandbox/invoke.go`, the worker creates a spool
  directory unique to that session and exposes its path to the spawned `claude`
  process through the agreed environment variable (e.g.
  `SPEKK_CONVERSATION_SPOOL`) — so the process's shell tools inherit it.
  **Note:** the variable must be scoped to that one spawned process, not set on
  the whole worker, so concurrent sessions get separate spools.
- Request files that appear in the spool during the session are read, and for
  each the worker emits one `conversation_open` frame (via the constructor from
  `conversation-open-frame`) on the session's WebSocket connection.
- `session_id` on every emitted frame is the initiating Claude session id as the
  worker knows it — for a resumed session, the resumed id; for a fresh session,
  the id observed from that session's own event stream (the same source used for
  the `result` frame's `session_id`). The value carried in the request file, if
  any, is never trusted for this.
- `title`, `body`, and `severity` come from the request file. A request missing
  a required field or carrying an out-of-range severity is dropped with a log
  line and does not send a frame and does not crash the worker.
- **Fire once, no buffering:** each request produces at most one frame. After a
  request is emitted its file is removed. There is no retry, no queue, and no
  offline buffering — if the WebSocket write fails, the failure is logged and
  the worker moves on.
- Requests are emitted promptly within the life of the session, not deferred
  past it: a request written mid-session is drained before the invocation
  returns. **Note:** the exact drain trigger (draining alongside the stdout
  scan loop vs. a small polling loop) is the implementer's choice; the
  observable requirement is that a request written during the session is emitted
  during that same invocation, and a request that arrives just before the
  process exits is still emitted (a final drain after the process ends).
- The spool directory is cleaned up when the session ends.
- A test drops a request file into a spool and drives the drain path, asserting
  that exactly one `conversation_open` frame is produced with the worker's
  session id (not one from the file), and that a malformed request produces no
  frame and no panic.
