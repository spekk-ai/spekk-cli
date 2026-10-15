---
id: stream-frames-name-their-session
created: 2026-10-15T18:31:03Z
priority: 1
---

# Stream Frames Name Their Session

## Problem

The sandbox client runs up to five sessions at once, but a `stream` frame carries only the output line. The control host can attribute it only to the turn it dispatched last. When two sessions stream at the same time, the lines of one can be attributed to the other.

## Solution

Each `stream` frame also carries the turn's `agent_session_id`, as the `result` and `error` frames already do. The field is additive, so `ProtocolVersion` does not change, and a control host that does not read it is not affected.
