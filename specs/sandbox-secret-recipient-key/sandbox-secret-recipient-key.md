---
id: sandbox-secret-recipient-key
created: 2026-10-23T11:00:00Z
priority: 1
---

# The agent client has a recipient key for secrets and shows its fingerprint

## Problem

The control host (spekk-app) will deliver a named secret, for example a package registry token, to a team sandbox that already runs. It seals the value to a public key that only that sandbox holds, so the control host stores no value that it can open. The contract is in spekk-app, in `specs/sandbox-secret-delivery/sandbox-secret-delivery.md` (section "Contract with spekk-cli") and `specs/sandbox-secret-delivery/assertions/secret-recipient-key.md`.

Today the agent client has no key pair, sends no public key, and ignores the `agent_id` field of `welcome`. The control host accepts a key only after an organization admin compares its fingerprint in the Secrets section. The admin has no fingerprint from the sandbox to compare it with.

This spec is the first client slice: the key pair, the `secrets_hello` frame, the agent id, and the fingerprint display. The handling of `secret_put` and `secret_remove`, the decryption, the write of a secret file, the `secret_ack` frame, the version state, and the redaction are the next slice.

## Solution

- **One key pair for each agent client home.** At start, the agent client reads its X25519 private key from `$XDG_CONFIG_HOME/spekk-agent/secret-key` (on a sandbox, `/home/agent/.config/spekk-agent/secret-key`). When the file is not there, it makes a new key pair and writes the file, mode 0600, in a directory with mode 0700. A restart uses the same key. A file with group or other permissions, or with contents that are not a key, is refused and kept: the agent logs the reason and sends no key. The agent never logs, prints, or sends the private key.
- **`secrets_hello` after each `welcome`.** The agent sends `{"type": "secrets_hello", "public_key": "<base64>", "held": {}}`. `held` is empty because this slice applies no secret. An old control host answers `Unknown message type`: the agent logs one time that the control host does not support secrets, and the connection continues.
- **The agent id.** The agent keeps the `agent_id` of `welcome`. The id for the seal is `SPEKK_AGENT_ID` when it is set, else that value. The next slice uses it.
- **The fingerprint.** The fingerprint is the first 16 hex characters of SHA-256 of the 32 public key bytes, the same value that the Secrets section of spekk-app shows. The agent logs it at each connect and writes it in the connection record (`/var/log/spekk/agent-connection.json`). `spekk sandbox status` shows it, in text and in the new `--json` output.

### Rejected

- **A key for each agent id from `welcome`.** The key file would then depend on a value that the agent gets only after a connect, and status could not show a fingerprint before the first connect. All agent ids on one box are the same `agent` user, so separate keys do not protect more. A new registration of the box keeps the key, and an admin must accept it again for the new agent.
- **Replace a bad key file with a new key.** A silent new key hides a fault or a change of the file, and it makes an admin accept a key that nobody expected. The agent refuses the file and a person decides.
- **A second SSH read of the public key for status.** The connection record already goes to status over SSH, and it is readable by the login user. The fingerprint is not a secret.
- **`golang.org/x/crypto` for this slice.** The Go standard library (`crypto/ecdh`) makes and reads X25519 keys. The seal of the next slice needs HKDF and ChaCha20-Poly1305.

## What this does not do

- No decryption, no secret file, no `secret_put`, `secret_remove`, or `secret_ack` handling, no version state, and no redaction. These are the next slice.
- No change to the control host and no protocol major change. `secrets_hello` is additive.
- No key rotation command. A person who removes the key file and restarts the agent gets a new key, and an admin must accept it.
