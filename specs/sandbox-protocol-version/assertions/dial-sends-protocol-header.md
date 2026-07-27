---
id: dial-sends-protocol-header
parent: sandbox-protocol-version
created: 2026-07-27T19:00:00Z
priority: 1
status: done
depends-on: protocol-version-constant
---

# Every Dial Sends the Protocol Header

## Description

The WebSocket dial carries `X-Spekk-Protocol: <ProtocolVersion>` next to the Authorization header, on every connection attempt.

## Success Criteria

- `dialOptions()` sets the `X-Spekk-Protocol` header to `ProtocolVersion`.
- The header is present on reconnects too (it is part of the dial options, not per-attempt state).

## Tests

- A test on `dialOptions()` asserts both headers: Authorization and X-Spekk-Protocol.
