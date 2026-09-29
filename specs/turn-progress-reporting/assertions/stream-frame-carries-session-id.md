---
id: stream-frame-carries-session-id
parent: turn-progress-reporting
created: 2026-08-08T04:30:00Z
priority: 1
status: not_started
branch: feature/turn-progress-reporting
---

# A Stream Frame Names the Session It Belongs To

## Description

The sandbox runs a pool of five workers keyed by `AgentSessionID`, so one connection multiplexes up to five concurrent sessions over one socket. A `stream` frame carries only `{"type": "stream", "data": <line>}` — no session id at all.

The control host advances a turn on every stream frame it receives, using the turn it dispatched most recently. With one session in flight that is right by luck. With two, every frame from the older session advances the newer session's turn. The older turn then stops advancing, ages past the silence bound, and is reported as stalled while its output is arriving; the newer turn reads as working on the strength of output that is not its own. Both halves of that are the exact failure the control host's lifecycle exists to remove, and no amount of care on the control host can fix it, because the information is not on the wire.

The keepalive frame already carries `agent_session_id` for this reason. The stream frame is the same problem and needs the same field.

## Success Criteria

- The `stream` frame carries `agent_session_id`, taken from the dispatch the worker is serving, alongside the existing `data` field.
- The field is added, not substituted. `data` keeps its meaning and its shape: one line of Claude's `stream-json` output, unmodified.
- Every `stream` frame a worker sends carries it, including frames drained after the Claude process exits. A frame with no session id is never emitted on a path that has one available.
- The field is additive, so the protocol minor moves and the major does not. It composes with `sandbox-protocol-version-1-1` rather than requiring a further bump.
- Tests cover: a frame emitted mid-invocation carries the dispatch's `agent_session_id`; two workers serving different sessions emit frames carrying their own session ids and never each other's.

**Note — rollout order.** Unlike `turn_keepalive`, this one is safe in either order: an added field on an existing frame is ignored by a control host that does not read it, and a control host that does read it must treat the field as optional until every client sends it. The control host change is therefore a follow-up, not a prerequisite. Until the control host reads the field it must keep advancing nothing rather than guessing when more than one turn is in flight, which is what it does today.
