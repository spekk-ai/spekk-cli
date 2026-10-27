# Spekk CLI 1.47.0 - App-Managed Sandboxes From the CLI, and a Key for Secrets

This release adds three changes. A new `spekk app` command family launches and uses app-managed sandboxes with a personal API key. The sandbox agent makes a recipient key for secrets and sends its public key to the control host. `create`, `provision`, and `register` now write `SPEKK_AGENT_ID` to the agent env file. The protocol version stays `1.5`.

## `spekk app`: app-managed sandboxes from a local CLI

A local developer, or a local agent such as Claude Code, can now launch a sandbox through the app API, send a text task, and read the result. These commands do not create a cloud machine directly and need no provider credential. The separate `spekk sandbox` commands still manage machines directly.

```bash
printf '%s\n' "$SPEKK_PERSONAL_KEY" | spekk app auth set --host https://app.spekk.ai
spekk app sandbox options
spekk app sandbox launch --organization ORG_ID --audience team --size SIZE --region REGION --kind one-off --repository REPO_ID
spekk app sandbox send --launch LAUNCH_ID --text 'Run the tests'
spekk app sandbox result --agent AGENT_ID --message MESSAGE_ID --timeout 10m
spekk app sandbox destroy --launch LAUNCH_ID
```

The subcommands are `options`, `repositories`, `agents`, `launch`, `list`, `status`, `wait`, `cancel`, `destroy`, `send`, `result`, and `operations`. Each command has `--json` for one JSON object.

- `spekk app auth set` reads the personal key from standard input. It keeps the key in `spekk/app/auth.json` in the user config directory, with mode 0600 in a 0700 directory. The CLI refuses a key file or a directory that other users can read.
- The host must use HTTPS. Plain HTTP works only for a loopback host. The CLI refuses redirects and sends the key only to the exact host.
- `launch` and `send` make a request ID (a UUID) and save it on the disk before the request. After a lost response, `status --request-id` and `result --request-id` read the saved state. The CLI does not start a second sandbox or send a task again automatically.
- The CLI warns until cleanup is complete: the launch stage is `failed` or `destroyed`, and `finished` is set. A local exit does not cancel a task or stop a machine.

The control host must support personal API keys and the launch and chat routes with request IDs. The server makes all access decisions. For the full setup, the recovery steps, and two example transcripts, see [App-managed sandboxes](../app-managed-sandboxes.md).

## The sandbox agent has a recipient key for secrets

This is the first client part of secret delivery to a running sandbox. In this release, the agent does not receive, decrypt, or write a secret.

- At start, the agent loads its X25519 private key from `/home/agent/.config/spekk-agent/secret-key`. When the file is not there, the agent makes a new key and writes the file with mode 0600 in a 0700 directory. A restart keeps the same key.
- The agent refuses a key file that other users can read, a symbolic link, or contents that are not a key. It keeps the file, makes no new key, logs the reason, and continues to serve turns. To repair it, fix or remove the file and restart the agent.
- After each `welcome`, an agent with a usable key sends a new `secrets_hello` frame with its public key. A control host that does not know the frame answers `Unknown message type`. The agent logs this one time and continues. The frame is additive, so the protocol version does not change.
- The agent keeps the `agent_id` of `welcome`. The agent ID for a secret is `SPEKK_AGENT_ID` when it is set, else that `agent_id`.
- The agent never logs, prints, or sends the private key. The image hygiene clean step and the `destroy` of a machine that stays remove the key directory.

**Secret delivery stays inactive.** The secrets feature of the control host is off until the app flag is on. While it is off, the control host ignores `secrets_hello` and sends no secret. This release also has no handler for the `secret_put` and `secret_remove` frames, and sends no `secret_ack`. The next client part adds them.

### `spekk sandbox status` shows the key fingerprint, and has `--json`

`spekk sandbox status <name>` prints a new line after the handshake lines:

```
Secret key fingerprint: 300c9c9603b92a4b
```

The fingerprint is the first 16 hex characters of SHA-256 of the public key. An organization admin compares it with the value in the Secrets section of the control host before accepting the key. For a refused key file, the line says `none (<reason>)`. A 1.46.0 agent binary gives `unknown (no fingerprint in the record; ...)`, and an older agent binary gives `unknown (no record; ...)`.

`spekk sandbox status <name> --json` prints one JSON object with the same values as the text lines, with snake_case keys, for example `agent_service` and `secret_key_fingerprint`. A version warning is in its own key, for example `agent_client_warning`.

## Each registered sandbox has `SPEKK_AGENT_ID` in its env file

The agent reads `SPEKK_AGENT_ID` from `/etc/spekk/agent.env` to ask the control host for repository access and for a Git token. Before this release, only the machine path wrote that line. On a sandbox from `create`, `provision`, or `register`, each chat turn that selected a repository failed with `Repository preparation failed: agent or session ID is missing`.

- `create` and `provision` write `SPEKK_AGENT_ID` with the agent ID of the record. A sandbox with no registered agent gets an empty value.
- `register` replaces the `SPEKK_AGENT_TOKEN` and `SPEKK_AGENT_ID` lines together and keeps every other line. If the restart fails, the script puts back the old file, with the old token and the old ID.

**An existing host does not get the line from this release.** `spekk sandbox deploy` does not change the env file, and `infrastructure/sandbox/setup-credentials.sh` does not write `SPEKK_AGENT_ID` yet. It keeps a line that is already in the file. To add the line to a sandbox that was equipped before 1.47.0, do one of these steps:

- Run `spekk sandbox register <name>`. This registers a new agent and revokes the old agent. The new agent starts with no schedules.
- Add one line `SPEKK_AGENT_ID=<agent id>` to `/etc/spekk/agent.env`, then restart the agent with `sudo systemctl restart spekk-agent`. The agent ID is the `agentId` field of the sandbox in `sandboxes.json`, in the spekk config directory (`$HOME/.config/spekk/` or `$XDG_CONFIG_HOME/spekk/`).

## Upgrading

The recipient key, the `secrets_hello` frame, and the fingerprint record are in the agent binary. After the release is published, use `spekk sandbox deploy <name>` on each authorized sandbox to install the new CLI and agent. Confirm the versions and the `Secret key fingerprint` line with `spekk sandbox status <name>`. Then add `SPEKK_AGENT_ID` to each existing host that needs repository chat, as described above.
