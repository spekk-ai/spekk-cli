# Spekk CLI 1.17.0 — The Sandbox States Its Protocol Version

The agent-client and the control host share a WebSocket contract, but ship separately. Until now, nothing stated or checked which contract a running pair speaks. This release gives the contract one version number and exchanges it at connect time.

## What the client does

- One constant declares the version: `ProtocolVersion = "1.0"`. A test pins it, so a change is always a deliberate, reviewed diff.
- Every dial sends `X-Spekk-Protocol: 1.0` next to the Authorization header.
- The control host replies with a `welcome` frame that carries its own version. Same major: a log line. Different major: a clear warning that names both versions and tells the operator to update the sandbox.
- When the control host refuses the version (close code 4004), the client logs one operator-facing line and keeps its normal reconnect backoff. It never hot-loops.

## Bump rules

A breaking change to message types, frame fields, or close codes bumps the major. An additive change bumps the minor. A PR that bumps the major names the companion control-host PR. The server owns enforcement; the client informs.

## Compatibility

Either side updates first, safely. An old client sends no header and the server accepts it as a legacy client. A new client against an old server sees no welcome frame and continues.
