---
id: sandbox-protocol-version-1-1
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 2
status: done
branch: feat/turn-progress-client
depends-on: result-frame-declares-terminal
---

# Protocol 1.1 Introduced the Terminal Result Field

## Description

Version `1.1` adds one field: `terminal` on the `result` frame. Under the bump rules already recorded on the version constant, that is an additive change: the minor moves and the major does not.

Version `1.1` introduced only the `terminal` field. The planned `turn_keepalive` frame takes the next unused minor version when it ships. A distinct version lets each side identify the frames that the other side supports.

## Success Criteria

- Protocol version `1.1` introduced the `terminal` result field. Later additive changes increase the minor version under the bump rules in `cmd/sandbox/protocol.go`.
- The protocol tests check that the connection header uses `ProtocolVersion`, so a later additive version keeps the declared version and the header consistent.
- The major-version comparison against the control host's `welcome` frame is unchanged. Both sides still speak major `1`, so no client warns and no connection is refused on account of this change.
- No breaking change is introduced. `terminal` is a new optional field; no existing frame loses a field, gains a required field, or changes meaning. A `1.1` client and a `1.0` control host work together, and a test shows that a `1.0` welcome does not make the client warn.
- The client and its docs reserve `turn_keepalive` for a later protocol version. Version `1.1` records only the `terminal` result field.
- The bump-rule comment on the constant is left intact, including the rule that a **major** bump names the companion control-host PR. That rule is not triggered here.

**Tests:** `cmd/sandbox/protocol_test.go`
