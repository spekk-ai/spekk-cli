---
id: turn-keepalive-frame-sent-during-tool-calls
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
status: not_started
branch: feature/turn-progress-reporting
depends-on: turn-inflight-tool-calls-tracked
---

# A Turn-Scoped Keepalive Reports Progress While a Tool Call Is in Flight

## Description

The control host derives abandonment from silence on a turn. That is only sound if silence means nothing is running. Today it does not: a long tool call is silent and healthy. This frame closes that gap, so silence keeps the meaning the control host's bound already assumes.

## Success Criteria

- A frame type `turn_keepalive` exists, declared as a constant alongside the other frame-type constants in `cmd/sandbox/message.go`.
- The frame carries `agent_session_id`, taken from the dispatch that opened the turn, so the control host can attribute it to one turn.
- The frame is sent **only** while at least one tool call is in flight. When the in-flight set is empty, no keepalive is sent, whatever else is happening.
- The frame is sent no more often than a named interval constant. The constant is separate from `heartbeatInterval` and its value is well inside the control host's 30-minute silence bound, so a single tool call lasting hours still reports progress. A comment records that it must stay comfortably below that bound and that the two constants are independent.
- No keepalive is sent when the turn is already producing output. A stream line arriving resets the interval, so a chatty turn adds no keepalive traffic at all and the frame appears only during genuine silence.
- Sending is best-effort. A write failure is not fatal to the turn: it does not end the invocation, does not raise, and does not change the outcome the turn will report. The keepalive is an observation about a turn, not part of its result.
- The keepalive is a distinct frame type. It is never emitted as a `stream` frame, because a `stream` frame is one line of Claude's output and a synthetic one would put text into the transcript that Claude never produced.
- The keepalive is independent of the agent heartbeat. The 30-second connection heartbeat continues unchanged, carries no turn identity, and is not repurposed here.
- Keepalives stop when the turn ends. None is sent after the `result` or `error` frame for that turn.
- Tests cover: a keepalive is emitted during a silent in-flight span; none is emitted when the set is empty; none is emitted while stream lines are flowing; a write failure leaves the turn's reported outcome unchanged; the frame carries the dispatch's `agent_session_id`.

**Note — rollout order is not negotiable.** This is a new frame type and it is not covered by the control host's absent-means-true guarantee. The control host must accept and attribute `turn_keepalive` **before** any client emits one. This project has already taken one production outage — 2026-08-07, on the control host — from shipping one side of a contract change ahead of the other. The companion change lives in the control host repository and is a dependency of this assertion's value, not of the code here.
