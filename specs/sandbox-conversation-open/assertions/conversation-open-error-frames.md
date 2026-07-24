---
id: conversation-open-error-frames
parent: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
status: not_started
---

# Inbound conversation_open Rejections Are Logged Legibly

When the control host rejects a `conversation_open` request it replies with a
normal `type: "error"` frame carrying a typed `error` code and a `detail`.
Today the worker's `readLoop` has no case for inbound `"error"` frames, so they
fall through to the `default` branch and log as `Unknown message type: error`,
which hides the actual code and detail. The worker should recognize these and
log them legibly.

## Success Criteria

- `cmd/sandbox/client.go`'s `readLoop` handles inbound frames of type `"error"`
  as their own `switch msg.Type` case rather than letting them hit the `default`
  (`Unknown message type`) branch. **Note:** a `MessageTypeError = "error"`
  constant already exists in `cmd/sandbox/message.go`; the new case matches on
  that constant (consistent with the existing `MessageTypeMessage` /
  `MessageTypeCancel` cases), not a bare literal.
- The inbound error frame's `error` code and `detail` are parsed and written to
  the log in a single legible line that includes both (e.g.
  `conversation_open rejected: conversation_open_no_channel — <detail>`).
  **Note:** the `Message` struct currently has no field for the error code or
  detail; add what is needed to read them.
- The three documented conversation_open codes are recognized specifically —
  `conversation_open_invalid`, `conversation_open_no_channel`,
  `conversation_open_failed` — so their log lines read as conversation_open
  rejections. Any other `error` code still logs its code and detail legibly
  rather than being swallowed.
- Receiving an error frame only logs; it does not tear down the connection or
  the worker (parity with how other non-fatal frames are handled today).
- A test feeds a `type: "error"` frame with a conversation_open code and detail
  through the inbound-handling path and asserts the log line contains both the
  code and the detail, and that the frame is not treated as an unknown type.
