---
id: result-frame-reaches-the-live-connection
parent: turn-survives-reconnect
created: 2026-08-24T16:50:00Z
priority: 1
status: done
depends-on: worker-turn-outlives-the-connection
---

# The `result` frame is delivered on whatever connection is live when it is sent

**Tests:** `cmd/sandbox/reconnect_test.go`

A turn that finishes after a reconnect reports its result on the new connection. A turn no longer holds the connection it started on.

## Success criteria

- `AgentClient` holds the current connection behind a mutex. `connect` publishes the connection after a successful dial and clears it when `connect` returns. A send resolves the current connection at the moment of sending.
- The worker no longer receives a `*websocket.Conn`. `Worker.Run`, `invoke`, `sendError`, and the `frameSender` passed to `drainSpool` all resolve the connection through the holder instead of closing over one.
- **The `result` frame waits for a live connection, up to 90 seconds.** If a connection becomes available within that time the frame is sent on it. If none does, the worker logs the loss with the session id and returns. The bound is longer than the 60-second worst case of `reconnectMax`, so a single **detected** drop never loses a result.
- Every frame that ends a turn names its turn. A final frame can arrive up to 90 seconds later, on a different connection than the dispatch, among as many turns as the pool runs, so an anonymous one cannot be attributed to any of them. The `result` frame already carried `agent_session_id`; the terminal error frames now carry it too. The field is additive and no existing field changed.
- The terminal error frame that `invoke` sends when `claude` exits non-zero waits the same way. It is the same report of a finished turn, and losing it leaves the control host with the same silence.
- **A `stream` frame does not wait.** With no live connection it is dropped and the turn continues. Streams drive a live display, so a late one has no value, and blocking on each of them would stall reading the child's stdout.
- A `conversation_open` frame drained from the spool does not wait either, and a send failure does not stop the drain or the turn. Its existing delivery semantics are unchanged.
- The wire format does not change. The same frame types carry the same fields, and `ProtocolVersion` is untouched.

- **A `stream` write is bounded.** The context a worker sends with is the process lifetime, so an unbounded write blocks for as long as the kernel retries a socket whose peer has stopped reading. That stalls the stdout scanner, fills the child's pipe, and stops `claude` itself — the opposite of what dropping a stream frame is for.

**Known limits.** Delivery is best-effort against an undetected drop, and this assertion does not claim otherwise. A write into a socket whose peer is gone but whose death the kernel has not yet reported returns success, so the result is lost and the log says the turn finished; only an application-level acknowledgement would close that, and none exists. Delivery is also at-least-once: a write that fails after its payload reached the peer is retried on the next connection, and no frame carries a nonce the control host could deduplicate on. Both predate this change and neither is fixed here.

**Note:** "waits for a live connection" means waiting for the holder to publish one, not dialing. `AgentClient.Run` owns reconnection. The worker must never dial.

**Note:** the send path is shared with a planned turn-progress effort that adds frames at these call sites. Keep the resolve-then-send helper small and obvious enough to add a frame type to.

## Test

In `cmd/sandbox/reconnect_test.go`, extend the connection-loss test from `worker-turn-outlives-the-connection`:

- Drop the connection mid-turn, let the client reconnect to a second test server, then let the child process finish.
- Assert the `result` frame arrives **on the second connection**, and that it carries the same `agent_session_id` and output it would have carried with no drop.
- Assert a `stream` frame produced during the gap is dropped rather than blocking the turn: the turn still reaches its result.

One test that covers delivery-after-reconnect and one that covers the drop-a-stream case are enough. Do not add a test per frame type.
