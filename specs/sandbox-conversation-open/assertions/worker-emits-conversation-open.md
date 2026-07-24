---
id: worker-emits-conversation-open
parent: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
status: in_progress
depends-on: conversation-open-frame
locked-by: builder-home-wsl2-648383-1784853848
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
  **Note (concrete plumbing):** `invoke.go` today constructs the command with
  `exec.Command("claude", …)` and never sets `cmd.Env` (a nil `cmd.Env` makes
  the child inherit `os.Environ()`). The spool variable must be added
  *per-process* by setting `cmd.Env = append(os.Environ(), "SPEKK_CONVERSATION_SPOOL="+spoolDir)`
  on that one command — **not** via `os.Setenv` on the worker, which is
  process-global and would make concurrent sessions share (and clobber) one
  spool. Appending to a copy of `os.Environ()` preserves the existing
  inheritance while scoping the new variable to the single spawned process. The
  variable name comes from the shared `conversation` package's env-var constant
  (`conversation-open-contract`), not a local literal.
- Request files that appear in the spool during the session are read, and for
  each the worker emits one `conversation_open` frame (via the constructor from
  `conversation-open-frame`) on the session's WebSocket connection.
- `session_id` on every emitted frame is the initiating Claude session id as the
  worker knows it. The worker tracks this id as the session runs: for a resumed
  session it is known up front from `msg.SessionID`; for a fresh session it is
  captured from the **earliest** stream event that carries a `session_id` (for
  current Claude `stream-json`, the initial `system`/`init` event). The value
  carried in the request file, if any, is never trusted for this. **Note:** the
  code today reads `session_id` only from the final `result` event — that is too
  late to stamp requests drained mid-session, so capture must also occur on the
  earlier event as the stream is scanned, seeding a session-id variable the
  drain path reads. This is the same underlying id the `result` frame ends up
  carrying; the change is *when* it is captured, not from where.
- If a request is drained before any session id is yet known (a fresh session
  whose id has not appeared in the stream), the request is dropped with a log
  line: no frame with an empty `session_id` is sent, and the request is not
  buffered for later delivery (consistent with `conversation-open-frame` and the
  spec's no-buffering scope). In practice the `init` event precedes any
  tool-invoked request, so the id is known by the first drain and this is a rare
  edge, not the common path.
- `title`, `body`, and `severity` come from the request file, decoded into the
  shared `conversation` package's request struct (`conversation-open-contract`)
  rather than an ad-hoc struct declared in the worker. A request missing a
  required field or carrying an out-of-range severity (checked via the shared
  validity helper) is dropped with a log line and does not send a frame and does
  not crash the worker.
- **Fire once, no buffering:** each request produces at most one frame. After a
  request is emitted its file is removed. There is no retry, no queue, and no
  offline buffering — if the WebSocket write fails, the failure is logged and
  the worker moves on.
- The spool is drained at concrete, dependency-free trigger points: **once
  after each line** the worker reads in the existing `claude` stdout scan loop
  (the same loop that already forwards `stream` frames and can capture the
  session id), and **once more after the process exits** (a final drain). This
  uses no filesystem watcher (`fsnotify`) and no separate polling goroutine —
  zero new dependencies. Each per-line drain runs *after* that line has been
  processed for a possible `session_id`, so the id captured from the `init`
  event is already available to stamp any request drained later in the session.
  The observable requirement: a request written mid-session is emitted before
  the invocation returns, and a request that lands just before the process exits
  is still emitted by the final drain.
- The spool directory is cleaned up when the session ends.
- A test drops a request file into a spool and drives the drain path, asserting
  that exactly one `conversation_open` frame is produced with the worker's
  session id (not one from the file), and that a malformed request produces no
  frame and no panic.
