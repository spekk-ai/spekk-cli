---
id: sandbox-docs-record-turn-progress-protocol
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 2
status: done
branch: feat/turn-progress-client
depends-on: sandbox-protocol-version-1-1
---

# The Sandbox Architecture Doc Records the Turn Progress Contract

## Description

`docs/advanced/sandbox-architecture.md` carries the frame tables that state the contract between the agent and the control host. Two additions belong there, and so does the reason the ordering of the rollout matters.

## Success Criteria

- The agent-to-control-host frame table does not list `turn_keepalive`, because the agent does not send it. A short section says that the frame is reserved for version `1.2`, what it will carry, and when the agent will send it: only while a tool call is in flight.
- The `result` row records the `terminal` field, that an absent field means `true`, and that the client always sends it explicitly.
- The document says that this agent declares `1.1`, that `1.1` adds only `terminal`, and that the bump is additive. The `welcome` example shows a control host at `1.0`, so that a reader sees that the two sides can declare different versions.
- The document states why the keepalive exists in one sentence a reader can act on: the agent-level heartbeat is connection-scoped and says nothing about a turn, so without a turn-scoped signal a long tool call is indistinguishable from a hang.
- The document records the rollout order — the control host accepts and attributes `turn_keepalive` before any agent emits it — and notes that the `terminal` field carries no such requirement because absent means `true`.
- The control-host responsibilities table gains the obligation to read `terminal` from the `result` frame. The keepalive obligation comes with `1.2`.
- Nothing in the document names the control host's implementation stack, its repository, a specific private hostname, or its internal admin URL structure, per `specs/sandbox-public-boundary/`.

**Done note (2026-10-15).** The `turn_keepalive` row and its control-host obligation were removed from the tables when `1.1` was limited to `terminal`. They come back with `1.2`.
