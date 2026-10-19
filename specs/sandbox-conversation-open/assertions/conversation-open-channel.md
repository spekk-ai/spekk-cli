---
id: conversation-open-channel
parent: sandbox-conversation-open
created: 2026-10-19T00:00:00Z
priority: 1
status: done
---

# Optional Conversation Channel

`spekk conversation open --channel <name|id>` writes an optional channel into the request file. The worker forwards it in the `conversation_open` frame only if the current server welcome frame advertises explicit channel support. Otherwise, the worker logs a clear refusal and sends no frame. The worker continues to stamp its own session ID. Without the flag, the frame omits the field. A `conversation_open_channel_rejected` response logs its code and detail clearly.

**Tests:** cmd/spekk/conversation_test.go, cmd/sandbox/spool_test.go, cmd/sandbox/client_test.go
