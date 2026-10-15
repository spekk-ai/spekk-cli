---
id: stream-frame-carries-agent-session-id
parent: stream-frames-name-their-session
created: 2026-10-15T18:31:03Z
priority: 1
status: done
---

# Each Stream Frame Carries the Turn's `agent_session_id`

**Tests:** `cmd/sandbox/invoke_test.go`

## Success criteria

- Every `stream` frame that `invoke` sends has `agent_session_id` set to the dispatched message's `agent_session_id`.
- The frame keeps its `type` and `data` fields unchanged, and `ProtocolVersion` does not change.
