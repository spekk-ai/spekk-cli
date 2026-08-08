---
id: turn-inflight-tool-calls-tracked
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
status: not_started
branch: feature/turn-progress-reporting
depends-on: claude-stream-shapes-pinned-by-fixtures
---

# The Client Knows, at Every Moment, Whether a Tool Call Is in Flight

## Description

A turn can be doing real work and printing nothing. Measured on Claude Code 2.1.226, a 20.6-second `Bash` call emitted zero stream lines between its start and its result. The client is the only party that can tell that window apart from a hang, because it is the only party that sees the bracketing events.

This assertion establishes the fact. Sending it to the control host is a separate assertion.

## Success Criteria

- The client maintains, per invocation, the set of tool-call ids that have started and not yet ended.
- A decoded tool-call start adds its `id` to the set. A decoded tool-call end removes the id matching its `tool_use_id`.
- Pairing is by id only. Neither counting nor ordering is used, because several tool calls may be in flight at once and their results may return in any order.
- A tool-call end whose `tool_use_id` is not in the set leaves the set unchanged. It is not an error and it never makes the set negative.
- "Work is in flight" is exactly "the set is non-empty". It is a derived answer read from the set, never a separately stored flag that a missed event could leave stale.
- The set is scoped to one invocation and starts empty. Nothing from a previous turn can make a new turn look busy.
- When the stream ends, the set is discarded. A tool call still open at stream end does not leak into the next turn.
- No inspection of a tool's name, its input, or any text decides whether work is in flight. Only the presence of unmatched ids does.
- Tests cover: one call in flight then closed; two overlapping calls closing in reverse order; a result for an unknown id; a stream that ends with a call still open. A fixture replay asserts the set is non-empty across the whole silent span of the long foreground tool call.
