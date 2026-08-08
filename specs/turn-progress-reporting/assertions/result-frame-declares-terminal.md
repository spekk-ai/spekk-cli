---
id: result-frame-declares-terminal
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
status: not_started
branch: feature/turn-progress-reporting
depends-on: turn-background-task-inventory-tracked
---

# The Result Frame Declares Whether the Task Ended or Only the Turn Did

## Description

The control host already reads a `terminal` field from the `result` frame and treats an absent field as `true`. Nothing sends it, so `YIELDED` is unreachable and every yielding turn records as `COMPLETED`. This assertion makes the client send it, decided from the background-task inventory rather than from anything the agent says about itself.

## Success Criteria

- The `result` frame carries a boolean field `terminal` alongside its existing `session_id`, `agent_session_id`, and `output` fields.
- `terminal` is `false` when the background-task inventory was non-empty at the **last** `result` event observed before the process exited. It is `true` otherwise.
- **The last `result` event is the one that counts, not the first.** A turn may emit more than one: measured, a turn that started an 8-second background task emitted a `result` while the task was live, waited, and emitted a second `result` once it finished. Evaluating the first would report a yield for a turn that genuinely completed. A comment records this, because taking the first result is the obvious and wrong reading.
- The field is always present on a `result` frame the client sends. The absent-means-true default belongs to the control host for the benefit of older clients; a client that implements this assertion states the value explicitly rather than relying on it.
- `terminal` is a real JSON boolean. It is never a string, never a number, and never null — the control host treats each of those as absent and logs a warning.
- A turn that starts no background work reports `terminal: true`, which is byte-for-byte the meaning today's client conveys by omitting the field. The ordinary case does not change behaviour.
- The `error` path is unchanged. A turn whose claude process exits non-zero still sends an `error` frame and no `result` frame, and `terminal` has no bearing on it.
- The value is never derived from the result text, the agent's reply, `subtype`, `stop_reason`, or `terminal_reason`. Only the inventory count decides it. In particular `terminal_reason: "completed"` was observed on a turn that had left a background task running and is not evidence of anything.
- Tests cover, by fixture replay: a turn with no background work reports `terminal: true`; a turn that ends with a live background task reports `terminal: false`; a turn whose background task finishes before a second `result` reports `terminal: true`. A test asserts the JSON encodes a bare boolean.

**Note — this direction is backward compatible, the keepalive is not.** The control host accepts an absent `terminal` as `true`, so shipping this ahead of anything else strands nobody. That guarantee covers this field only. It does not cover the new frame type in `turn-keepalive-frame-sent-during-tool-calls`, which has its own ordering requirement.
