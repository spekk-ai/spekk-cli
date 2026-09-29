---
id: sandbox-docs-record-turn-progress-protocol
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 2
status: not_started
branch: feature/turn-progress-reporting
depends-on: sandbox-protocol-version-1-1
---

# The Sandbox Architecture Doc Records the Turn Progress Contract

## Description

`docs/advanced/sandbox-architecture.md` carries the frame tables that state the contract between the agent and the control host. Two additions belong there, and so does the reason the ordering of the rollout matters.

## Success Criteria

- The agent-to-control-host frame table lists `turn_keepalive`, its `agent_session_id` field, and the condition under which it is sent: only while a tool call is in flight.
- The `result` row records the `terminal` field, that an absent field means `true`, and that the client always sends it explicitly.
- The protocol version stated in the document reads `1.1`, and the bump is described as additive.
- The document states why the keepalive exists in one sentence a reader can act on: the agent-level heartbeat is connection-scoped and says nothing about a turn, so without a turn-scoped signal a long tool call is indistinguishable from a hang.
- The document records the rollout order — the control host accepts and attributes `turn_keepalive` before any agent emits it — and notes that the `terminal` field carries no such requirement because absent means `true`.
- The control-host responsibilities table gains the two matching obligations: attribute `turn_keepalive` to a turn, and read `terminal` from the `result` frame.
- Nothing in the document names the control host's implementation stack, its repository, a specific private hostname, or its internal admin URL structure, per `specs/sandbox-public-boundary/`.
