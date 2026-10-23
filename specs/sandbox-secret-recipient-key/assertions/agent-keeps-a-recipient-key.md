---
id: agent-keeps-a-recipient-key
parent: sandbox-secret-recipient-key
created: 2026-10-23T11:00:00Z
priority: 1
status: done
---

# The agent client keeps one X25519 key pair over restarts and refuses a key file that is not safe

**Tests:** `cmd/sandbox/secret_key_test.go`

## Success criteria

- At start, the agent client loads its private key from `$XDG_CONFIG_HOME/spekk-agent/secret-key`, else `$HOME/.config/spekk-agent/secret-key`. On a sandbox this is `/home/agent/.config/spekk-agent/secret-key`.
- When the file is not there, the agent makes a new X25519 key pair (`crypto/ecdh`), makes the directory with mode 0700, and writes the file with mode 0600. The file holds the 32 private key bytes as standard base64 and a newline. The write uses a temporary file and a link, so a reader never sees a part of the key and the write never replaces a file that another process wrote first.
- Before it makes a new key, the agent refuses the key directory when the directory has a group or other write bit (for example, mode 0777) or the agent user does not own it. The agent does not change the mode of a directory that is there.
- A restart loads the same key, so the public key and the fingerprint do not change.
- The agent refuses the file, keeps it, and makes no new key when:
  - the file is not a regular file (for example, a symbolic link, a directory, or a FIFO). The open does not wait for a writer, so a FIFO at the key path does not stop the agent,
  - the file has a group or other permission bit (for example, mode 0644),
  - the agent cannot read the file, or
  - the contents are not standard base64 of 32 bytes.
- After a refusal, the agent logs one line with the path and the reason, and continues to serve turns. It sends no `secrets_hello` until a person fixes or removes the file and restarts the agent.
- No log line, error text, frame, or status output contains the private key bytes, in base64 or in any other form.
- The key file is in the clean step of the image hygiene script and in the files that `destroy` removes from a machine that stays after destroy, so a template image or a machine for a new agent never carries the key of an old sandbox.

## Test

Real files in a temporary directory: a first load makes a 0600 file in a 0700 directory, and a second load gets the same public key. A 0644 file, a file with bad contents, a symbolic link, a FIFO, and a missing file in a 0777 directory are refused, are not changed, no new key is made, and the error text does not contain the key.
