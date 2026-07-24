---
id: conversation-open-frame
parent: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
status: in_progress
depends-on: conversation-open-contract
locked-by: builder-home-wsl2-644046-1784853646
---

# Worker Builds a Well-Formed conversation_open Frame

The worker has a single, typed way to construct the `conversation_open` frame
it sends to the control host, so the shape and its encoding rules live in one
place rather than being hand-assembled at each call site.

## Success Criteria

- A named message-type constant for `"conversation_open"` exists alongside the
  other `MessageType*` constants in `cmd/sandbox/message.go` (e.g.
  `MessageTypeConversationOpen`). No literal string `"conversation_open"` is
  duplicated across call sites.
- A single constructor/serializer produces the outgoing frame with exactly
  these fields: `type` (the constant), `session_id`, `title`, `body`,
  `severity`, and `metadata`. **Note:** `conversation_open` is worker→control-host
  only; it must **not** be added as a field or case to the inbound `Message`
  struct or `readLoop`. Existing outbound frames (`stream`, `result`, `error`)
  are hand-assembled as `map[string]any` at their call sites; this constructor
  centralizes the one frame with encoding rules worth pinning, and is written
  over the WebSocket the same way (e.g. via `wsjson.Write`).
- `session_id` is always populated with a non-empty value in a frame that gets
  sent. **Note:** the contract requires `session_id`; the constructor (or its
  caller) must not emit a frame with an empty `session_id` — if the session id
  is not yet known, no frame is sent and the condition is logged (see
  `worker-emits-conversation-open`).
- `severity` is constrained to exactly one of `info`, `warning`, `critical`.
  An absent or empty severity serializes as `info` (the default). A value
  outside the three is not sent as-is — it is rejected or coerced, never passed
  through verbatim. **Note:** the severity constants, the `info` default, and the
  validity check come from the shared `conversation` package
  (`conversation-open-contract`); this constructor does not re-declare its own
  severity value set.
- `metadata` is optional. When there is no metadata, the field is omitted from
  the serialized JSON (or sent as an empty object) rather than sent as `null`.
  **Note:** pick one representation and keep it consistent; the point is that
  "no metadata" never serializes to the literal `null`.
- A unit test constructs a frame and asserts: the `type` value, that a valid
  severity round-trips, that empty severity becomes `info`, that an invalid
  severity does not pass through, and that empty metadata is omitted (or empty)
  rather than `null`.
