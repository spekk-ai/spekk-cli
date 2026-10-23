---
id: status-shows-the-key-fingerprint
parent: sandbox-secret-recipient-key
created: 2026-10-23T11:00:00Z
priority: 1
status: not_started
depends-on: agent-keeps-a-recipient-key
---

# `spekk sandbox status` shows the fingerprint of the recipient key, in text and in JSON

**Tests:** `internal/agentconn/agentconn_test.go`, `cmd/sandbox/secret_key_test.go`

## Success criteria

- At start, the agent writes the result of the key load in the connection record (`/var/log/spekk/agent-connection.json`): `secret_key_fingerprint` for a usable key, or `secret_key_error` with the reason for a refused key. The record never holds a key.
- `spekk sandbox status <name>` prints `Secret key fingerprint: <16 hex characters>` after the handshake lines. For a refused key it prints `none (<reason>)`. For a record with neither field it prints `unknown (no fingerprint in the record; the agent-client may be older than this check)`. For no record, an SSH failure, or a record that is not valid, it prints the same `unknown` values as the handshake lines.
- `spekk sandbox status <name> --json` prints one JSON object with the same values as the text lines, with snake_case keys, and the fingerprint as `secret_key_fingerprint`. The text output does not change, other than the new line.
- The fingerprint in the record is the same value that the agent logs and that spekk-app computes: the first 16 hex characters of SHA-256 of the 32 public key bytes.

## Test

A fixed test vector: a fixed private key, with the fingerprint computed in the test from the public key bytes, is the fingerprint that the agent writes. A table test of the status value for a record with a fingerprint, with an error, with neither, and with no record.
