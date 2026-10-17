---
id: sandbox-protocol-version-1-1
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 2
status: done
branch: feat/turn-progress-client
depends-on: result-frame-declares-terminal
---

# The Client Declares Protocol 1.1

## Description

Version `1.1` adds one field: `terminal` on the `result` frame. Under the bump rules already recorded on the version constant, that is an additive change: the minor moves and the major does not.

Version `1.1` means the `terminal` field and nothing else. The `turn_keepalive` frame type is reserved for `1.3`. If `1.1` also named the keepalive, a later client that sends it and a control host that does not accept it would declare the same version, and the version would not tell them apart.

## Success Criteria

- `ProtocolVersion` in `cmd/sandbox/protocol.go` reads `"1.1"`.
- The test that pins the constant's value is updated to `"1.1"`, so the change stays a deliberate, reviewed diff.
- The major-version comparison against the control host's `welcome` frame is unchanged. Both sides still speak major `1`, so no client warns and no connection is refused on account of this change.
- No breaking change is introduced. `terminal` is a new optional field; no existing frame loses a field, gains a required field, or changes meaning. A `1.1` client and a `1.0` control host work together, and a test shows that a `1.0` welcome does not make the client warn.
- Nothing in the client or its docs says that `1.1` includes `turn_keepalive`. The docs say that the frame is reserved for `1.3`.
- The bump-rule comment on the constant is left intact, including the rule that a **major** bump names the companion control-host PR. That rule is not triggered here.

**Tests:** `cmd/sandbox/protocol_test.go`
