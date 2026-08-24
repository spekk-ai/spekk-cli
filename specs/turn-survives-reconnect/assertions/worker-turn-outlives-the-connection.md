---
id: worker-turn-outlives-the-connection
parent: turn-survives-reconnect
created: 2026-08-24T16:50:00Z
priority: 1
status: done
---

# A lost connection does not signal the running `claude` process

**Tests:** `cmd/sandbox/client_test.go`

The worker's lifetime is the process, not the connection. When the WebSocket drops, the `claude` process a worker started keeps running, and the worker keeps reading its output.

## Success criteria

- `AgentClient` holds the process context that `Run` receives, and `handleMessage` passes **that** context to `w.Run`, not the per-connection context that `connect` derives. `connect` keeps its own derived context for the heartbeat and the read loop, which must still stop when the connection ends.
- The shutdown goroutine in `invoke` (`cmd/sandbox/invoke.go:73-80`) therefore fires only when the process context is done. Its behavior on a real shutdown is unchanged: `main` builds the context with `signal.NotifyContext` for SIGTERM and SIGINT, and a turn still takes SIGTERM when the process is asked to stop.
- `pool.Cancel` is unaffected. A `cancel` frame from the control host still signals the named session's process through `Worker.Cancel`.
- The pool is already owned by the client rather than by a connection, so a worker stays registered in `p.sessions` across a reconnect and a later message for the same session still routes to it. Confirm this holds; do not rebuild it.

**Note:** the fix is a change of which context is passed, not a new lifetime mechanism. Resist adding a second context layer, a worker-owned `context.WithCancel`, or a shutdown registry. One context, chosen correctly, is the whole change.

## Test

A test in `cmd/sandbox/client_test.go` that drives a real turn across a connection loss:

- Start a test WebSocket server. Dispatch a message that starts a long-lived child process. Close the server connection abnormally while that process runs.
- Assert the child process is **still running** after the close, and that it exits on its own terms rather than by signal.
- Assert the opposite case still works: when the process context is cancelled, the child is signaled.

The test must exercise the real `AgentClient` dispatch path. A test that calls `invoke` directly with a hand-made context proves nothing about which context the client passes, which is the entire defect.
