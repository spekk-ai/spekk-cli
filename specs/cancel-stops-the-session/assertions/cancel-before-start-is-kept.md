---
id: cancel-before-start-is-kept
parent: cancel-stops-the-session
created: 2026-10-23T05:00:00Z
priority: 1
status: done
depends-on: cancel-drains-queued-turns
---

# A cancel before the process starts stops the turn

**Tests:** `cmd/sandbox/cancel_test.go`

## Success criteria

- A cancel that arrives after the runner takes a message and before `invoke` sets `w.current` is recorded on the worker.
- `invoke` reads the record and starts the process under the worker mutex in one critical section. If the record is set, the process does not start, and the turn ends with a terminal `error` frame that names the session. If the record is not set, the process starts and `w.current` is set before the mutex is released, so a later cancel finds the process and signals it.
- The record applies only to the turn that is in flight at the time of the cancel. The runner clears it when it takes the next message, so a message that arrives after the cancel runs.

## Test

A real test through the real `AgentClient` dispatch path. A test hook in `invoke`, called just before the check, blocks the turn in the window. The test cancels the session while the hook blocks, queues one more message on the same worker, then releases the hook. Assert that the host received an `error` frame for the cancelled turn and a `result` frame for the later message, and that the fake `claude` ran exactly once. The test uses no sleep to find the window.
