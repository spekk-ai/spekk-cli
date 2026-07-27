---
id: protocol-version-constant
parent: sandbox-protocol-version
created: 2026-07-27T19:00:00Z
priority: 1
status: done
---

# The Client Declares One Protocol Version Constant

## Description

The protocol version lives in one Go constant. Nothing else in the client states a version.

## Success Criteria

- A constant `ProtocolVersion = "1.0"` exists in `cmd/sandbox` (for example `protocol.go`), with a comment stating the bump rules: breaking change to message types, frame fields, or close codes bumps the major; additive change bumps the minor; a major bump names the companion spekk-app PR.
- A helper returns the major part (`"1"` from `"1.0"`); malformed input returns the whole string (never panics).
- A test pins the constant's value, so a change is always a deliberate, reviewed diff.

## Tests

- Constant value pinned; major-part helper covered for `"1.0"`, `"2.3"`, and a malformed value.
