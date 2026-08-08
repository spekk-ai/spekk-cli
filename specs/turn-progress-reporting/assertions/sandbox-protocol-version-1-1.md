---
id: sandbox-protocol-version-1-1
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 2
status: not_started
branch: feature/turn-progress-reporting
depends-on: result-frame-declares-terminal
---

# The Client Declares Protocol 1.1

## Description

This spec adds one frame type and one optional field. Under the bump rules already recorded on the version constant, that is an additive change: the minor moves and the major does not.

## Success Criteria

- `ProtocolVersion` in `cmd/sandbox/protocol.go` reads `"1.1"`.
- The test that pins the constant's value is updated to `"1.1"`, so the change stays a deliberate, reviewed diff.
- The major-version comparison against the control host's `welcome` frame is unchanged. Both sides still speak major `1`, so no client warns and no connection is refused on account of this change.
- No breaking change is introduced anywhere in this spec. `turn_keepalive` is a new frame type and `terminal` is a new optional field; no existing frame loses a field, gains a required field, or changes meaning. A client at `1.0` and a control host at `1.1` interoperate exactly as they do today.
- The bump-rule comment on the constant is left intact, including the rule that a **major** bump names the companion control-host PR. That rule is not triggered here.
