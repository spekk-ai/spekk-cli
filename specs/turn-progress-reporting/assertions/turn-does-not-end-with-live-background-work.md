---
id: turn-does-not-end-with-live-background-work
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 3
status: not_started
branch: feature/turn-progress-reporting
depends-on: result-frame-declares-terminal
---

# A Turn Never Reports Success While Work It Started Is Unfinished

## Description

Reporting a yield accurately is second best. The end state is one where a yield cannot happen: turn-end means task-end by construction, and `COMPLETED` is honest without anyone trusting anything. Under this assertion `terminal: false` becomes a transitional path that production stops exercising, and the control host's `YIELDED` count trends to zero.

This is the goal, deliberately ranked behind the reporting work. Ship the reporting first — a `YIELDED` count is the only way to know whether this assertion is working.

## Success Criteria

- A turn that reaches its last `result` event with a non-empty background-task inventory does not report `terminal: true`, and does not report a `result` frame that the control host would record as `COMPLETED`.
- Such a turn resolves rather than reports. The client continues the turn — the session resume it already uses is the available lever — until the background-task inventory is empty, then reports the resolved outcome.
- Resolution is bounded by a named constant. When the bound expires with work still live, the turn reports `FAILED` via an `error` frame naming the number of unfinished background tasks. An unresolvable turn is a failure, not a success and not a silent yield.
- The bound is a distinct named constant with a comment recording that it caps how long one dispatch may be extended, and that expiry is reported as a failure rather than swallowed.
- No prose is read to decide whether to continue. Only the inventory count does, exactly as in `result-frame-declares-terminal`.
- The agent is never asked whether it is finished, and no text is appended to the prompt asking it to confirm completion. Self-assessment is the failure mode this spec exists to remove, and it must not re-enter through the resolution path.
- Once this holds, a turn reports exactly one of: `COMPLETED` with an empty inventory, or `FAILED`. `terminal: false` remains implemented and remains correct, and stops occurring in practice.
- Tests cover: a turn whose background task finishes within the bound reports `terminal: true`; a turn whose background task is still live when the bound expires reports `error` naming the count; a turn with no background work is unaffected and takes no extra time.

**Note — what the client cannot do.** A `claude -p` invocation is one-shot: the prompt is written to stdin, stdin is closed, and the turn cannot be extended in place. The client also cannot stop Claude Code from killing its own background tasks — measured, a task was killed 5 seconds after the turn reported success, so by the time the process exits the inventory has already emptied itself. Both facts mean resolution must be decided from the inventory observed **at the result event** and acted on by continuing the session, not by watching the process afterwards. An implementation that waits on the process tree will observe nothing.
