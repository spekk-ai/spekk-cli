---
id: sandbox-protocol-version
created: 2026-07-27T19:00:00Z
priority: 1
---

# Sandbox Protocol Version (Client Side)

## Problem

The agent-client and the control host share a WebSocket contract: message types, frame fields, close codes. The two ship from separate repositories on separate cadences, and nothing states or checks which contract a running pair speaks. Version skew is invisible until a frame is silently misunderstood.

## Decision

The contract gets one version number, declared on both sides and exchanged at connect time. The client sends its version in an `X-Spekk-Protocol` header on dial. The server replies with its own version in a `welcome` frame. Each side checks the other's **major** version; the server enforces (companion spec: spekk-app `protocol-handshake`), the client warns. The version starts at `1.0`.

Bump rules: a breaking change to message types, frame fields, or close codes bumps the major. An additive change bumps the minor. A PR that bumps the major names the companion PR in the other repository — this pairing is the one rule that stays human.

## Rejected

- Automatic cross-repo version sync. No mechanism can force two repositories to bump together; the handshake makes a missed bump loud instead.
- Feature-flag negotiation per capability. One number is enough at this scale.
- Client-side hard refusal on mismatch. The server owns enforcement; a client that refused to connect could never be told to update.
