---
id: client-handles-welcome-and-reject
parent: sandbox-protocol-version
created: 2026-07-27T19:00:00Z
priority: 1
status: not_started
depends-on: protocol-version-constant
---

# The Client Reads the Welcome Frame and Names a Protocol Reject

## Description

The client understands the two server-side signals: the `welcome` frame that advertises the server's protocol version, and close code 4004 that rejects an incompatible major.

## Success Criteria

- A `welcome` frame (`{"type": "welcome", "protocol": "<version>"}`) is handled: same major → a debug-level log; different major → a clear warning naming both versions and telling the operator to update the sandbox. The client stays connected either way — the server owns enforcement.
- A connection closed with code 4004 logs one clear line: the protocol was rejected, this sandbox must update. The normal reconnect backoff continues unchanged; the client never hot-loops on a 4004.
- A `welcome` frame is never treated as an unknown message type.

## Tests

- `handleInbound` with a welcome frame: no unknown-type log; major mismatch produces the warning.
- The 4004 close path produces the operator-facing log line (exercise the error-classification helper, not a live socket).
