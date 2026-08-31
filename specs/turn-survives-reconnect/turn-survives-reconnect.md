---
id: turn-survives-reconnect
created: 2026-08-24T16:50:00Z
priority: 1
---

# A dropped WebSocket must not kill the turn it was carrying

## Problem

The sandbox agent client keeps one WebSocket connection to the control host, and it reconnects on its own when that connection drops. A drop is ordinary: it happens several times a day, at irregular times, for network reasons outside this program. Almost every drop lands while the agent is idle, so nothing is lost.

When a drop lands while a turn is running, the turn dies. Three lines in the current source produce that outcome together:

- `cmd/sandbox/client.go:89-90` — `connect` derives a per-connection context and defers `cancel()`.
- `cmd/sandbox/client.go:183` — `handleMessage` gives that per-connection context to the worker goroutine.
- `cmd/sandbox/invoke.go:73-80` — `invoke` runs a goroutine, commented "SIGTERM on shutdown", that signals the `claude` process when that context is done.

So when `readLoop` returns on a dropped connection, `connect` returns, the deferred `cancel()` fires, and every in-flight `claude` process takes SIGTERM. The client then reconnects correctly, but the work is already dead. The comment names the intent exactly: the goroutine is meant to fire on process shutdown. The context it watches is the connection.

There is a second fault on the same path. `invoke` captures `conn` and writes every frame to it, including the final `result` frame (`cmd/sandbox/invoke.go:156`). A turn that outlived the drop would still have no way to report, because it holds the closed connection.

There is a third fault, in the reconnect loop itself. `Run` declares `delay := reconnectBase` **outside** the `for` loop and never resets it after a connection succeeds (`cmd/sandbox/client.go:60-76`). The backoff therefore ratchets across the whole life of the process: after about five drops, every later reconnect waits the full `reconnectMax` of 60 seconds, even when the connection before it was healthy for hours. This is visible in production, where every observed reconnect gap was exactly 60 seconds and never the 3 seconds a fresh backoff would give.

## Evidence

Two scheduled runs failed in production this way, on 2026-08-18 and on 2026-08-24. Both left no reply of any kind — no result, no error, no partial output. They are the only two failures of that shape on record, and the shape is what identifies them: a turn that ends by any normal path leaves a `result` frame or an error frame behind.

Exposure is proportional to how long a turn runs, so the longest job fails first. Both failures were long turns.

## Solution

Scope the turn to the process, and scope the send to whatever connection is live at the moment of sending.

1. The worker receives the **process** context, so only a real shutdown signals `claude`.
2. Frames go through a holder that resolves the current connection when the frame is sent, rather than a connection captured when the turn began.
3. The `result` frame — the one the control host acts on — waits a bounded time for a live connection, so it survives the reconnect gap.
4. The reconnect backoff resets after a connection that was established successfully.

## What this deliberately does not do

- **No outbound queue, and no persistence.** The `result` frame is the only frame the control host acts on. A bounded wait delivers it. A general queue with replay is machinery this problem does not need.
- **No replay of `stream` frames.** They drive a live display. A frame that missed its moment has no value, so a stream frame with no connection is dropped and the turn continues.
- **No control-host change, and no protocol change.** The same frames go over the wire, in the same shape, at the same protocol version. This ships by updating the sandbox binary alone. A control-host change would make this a two-sided contract, which is the hazard this repository already paid for once.
- **No reconnection logic in the worker.** `AgentClient.Run` owns reconnection and keeps it.
- **No attempt to detect a stalled turn.** That is a separate failure mode with its own spec.

## Interaction with turn progress reporting

A separate effort adds a turn-scoped keepalive frame, whose contract is that silence on a turn means nothing is running. After this spec is built, a turn stays alive across a reconnect gap of up to 60 seconds during which it cannot send anything. That silence is not a stalled turn, and a keepalive reader that treats it as one produces exactly the false stall it exists to prevent. Whoever builds the keepalive must account for the gap.

That effort must also build on the send path this spec leaves behind: it adds frames at the same call sites that stop capturing `conn` here.
