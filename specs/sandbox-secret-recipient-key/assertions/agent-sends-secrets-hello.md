---
id: agent-sends-secrets-hello
parent: sandbox-secret-recipient-key
created: 2026-10-23T11:00:00Z
priority: 1
status: not_started
depends-on: agent-keeps-a-recipient-key
---

# The agent client sends its public key in `secrets_hello` after each `welcome` and keeps the agent id

**Tests:** `cmd/sandbox/secret_key_test.go`

## Success criteria

- After each `welcome`, an agent with a usable key sends `{"type": "secrets_hello", "public_key": "<base64>", "held": {}}`. `public_key` is standard base64 with padding (RFC 4648 section 4) of the 32 public key bytes. `held` is an empty object, because this slice applies no secret.
- An agent with a refused key sends no `secrets_hello`.
- At each `welcome`, the agent logs one line with the fingerprint (the first 16 hex characters of SHA-256 of the 32 public key bytes), for the comparison in the Secrets section of spekk-app.
- The agent keeps the `agent_id` of `welcome`. The agent id for the seal is `SPEKK_AGENT_ID` when it is set, else the `agent_id` of the last `welcome`.
- An old control host answers `{"type": "error", "message": "Unknown message type: secrets_hello"}`. The agent logs one time for each process that the control host does not support secrets. The connection stays open, and the agent continues to read and answer frames. A later `welcome` sends `secrets_hello` again, so a control host that gets the feature later gets the key at the next connect.
- The protocol version does not change: `secrets_hello` is an additive frame.

## Test

The real `AgentClient.Run` against a test WebSocket server. The server sends `welcome` with an `agent_id`, reads `secrets_hello`, and checks the public key against the key file. It answers `Unknown message type`, then sends a second `welcome`, and gets a second `secrets_hello` on the same connection. The log has the "not supported" line one time and does not contain the private key.
