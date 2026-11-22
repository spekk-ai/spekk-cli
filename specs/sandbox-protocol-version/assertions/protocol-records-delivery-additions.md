---
id: protocol-records-delivery-additions
parent: sandbox-protocol-version
created: 2026-11-22T00:00:00Z
priority: 2
status: not_started
---

# The Protocol Version Records the File Delivery Additions

## Description

Spekk CLI 1.51 added the `agent_conversation_delivery` and `agent_current_thread_delivery` welcome capabilities and the optional `agent_turn_id` field on the `message` frame, and kept `ProtocolVersion` at `1.6`. Each is an addition, so the contract needs a minor bump.

## Success Criteria

- `ProtocolVersion` in `cmd/sandbox/protocol.go` is `1.7`, and its comment states that a new welcome capability or a new optional frame field is an addition that bumps the minor.
- `docs/advanced/sandbox-architecture.md` has a `1.7` entry that names the two capabilities and the `agent_turn_id` field, and the `message` frame example and the inbound frame table list `agent_turn_id`.
- A `1.6` control host still works with this client: the client uses each delivery feature only when the `welcome` frame advertises its capability.

## Tests

- The existing protocol and capability tests in `cmd/sandbox` pass with the new value.
