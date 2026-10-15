---
id: error-frame-carries-the-api-error
parent: turn-failure-reports-its-cause
created: 2026-10-15T16:30:00Z
priority: 2
status: done
---

# The error frame of a failed turn carries the API error from the `result` event

**Tests:** `cmd/sandbox/invoke_test.go`

## Success criteria

- When `claude` exits non-zero and its last `result` event has `is_error: true`, the `error` field is `claude exited: <exit status>: <result text>`.
- The `Claude failed` log line on the sandbox shows the same text.
- The added text is capped at 500 characters.
- When the last `result` event does not have `is_error` set, or there is no `result` event, the `error` field stays `claude exited: <exit status>`. A result that is not an error is the turn's own output, not a cause.
- The frame keeps its other fields (`detail`, `agent_session_id`), and `ProtocolVersion` does not change.

## Test

A fake `claude` writes an `api_retry` event and a `result` event with `is_error: true`, then exits 1. Assert that the error frame keeps the exit status and contains the API error text. A unit test of `failureReason` asserts that a result without `is_error` adds nothing.
