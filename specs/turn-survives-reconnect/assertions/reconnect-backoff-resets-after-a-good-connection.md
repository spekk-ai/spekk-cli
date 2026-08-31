---
id: reconnect-backoff-resets-after-a-good-connection
parent: turn-survives-reconnect
created: 2026-08-24T16:50:00Z
priority: 1
status: done
---

# The reconnect backoff resets after a connection that succeeded

**Tests:** `cmd/sandbox/reconnect_test.go`

The backoff measures how long the control host has been unreachable. It must not measure how long this process has been alive.

## Success criteria

- In `AgentClient.Run` (`cmd/sandbox/client.go:59-77`), `delay` resets to `reconnectBase` after a connection that was established successfully. Today it is declared before the loop and only ever doubles, so a process that has dropped about five times waits the full `reconnectMax` of 60 seconds for every later reconnect — including the first drop after hours of health.
- "Established successfully" means the connection **lived**, not merely that the dial returned. A connection that ends within `establishedMin` does not reset the backoff. Counting a returned dial as success defeats the backoff for the case that needs it most: a control host that accepts and closes at once — which is exactly the documented protocol reject, close 4004 — would be dialed every 3 seconds for the length of a deploy gap, where the previous code backed off to 60 seconds.
- A dial that failed must still back off, so a control host that is down is not dialed every 3 seconds.
- The growth path is unchanged: consecutive failures still double from `reconnectBase` up to `reconnectMax`.

**Note:** this is the observed production behavior, not a theoretical one. Every reconnect gap on record was exactly 60 seconds, never the 3 seconds a fresh backoff gives. Shortening the gap also shortens the window that the `result` frame in `result-frame-reaches-the-live-connection` must wait through.

## Test

A test in `cmd/sandbox/reconnect_test.go` that asserts the delay sequence, without sleeping for real:

- After a successful connection that then drops, the next wait is `reconnectBase`.
- After consecutive dial failures, the waits double and stop at `reconnectMax`.

Make the delay observable rather than timing it. Extract the backoff decision into a small function the test calls directly, or have `Run` take an injectable wait, whichever is the smaller change to the existing code.
