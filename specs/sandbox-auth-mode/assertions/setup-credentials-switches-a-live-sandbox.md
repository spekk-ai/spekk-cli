---
id: setup-credentials-switches-a-live-sandbox
parent: sandbox-auth-mode
created: 2026-08-27T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-subscription-auth
---

# `setup-credentials.sh` Moves an Existing Sandbox Between Auth Modes

The credential script takes the same two auth modes, prompts only for the credential its mode needs, and writes an env file that carries no leftovers from the mode the droplet used before. Its usage header describes an invocation that works.

**Tests:** internal/sandbox/setup_credentials_test.go

## Success Criteria

- The script reads the mode from `SPEKK_AUTH_MODE`, defaulting to `bedrock`. It rejects any other value with a message naming both accepted modes, and exits non-zero before touching `/etc/spekk/agent.env`.
- It prompts for the credential the mode needs and not the other: `subscription` asks for the Claude subscription token and never for an Anthropic API key; `bedrock` asks for the AWS credentials. Either can still be supplied non-interactively through the environment, as the other variables already can.
- `bedrock` mode writes the same model-credential block the Go path writes — the AWS trio and `CLAUDE_CODE_USE_BEDROCK=1`. It no longer writes `ANTHROPIC_API_KEY`.
- `subscription` mode writes `CLAUDE_CODE_OAUTH_TOKEN` and none of `CLAUDE_CODE_USE_BEDROCK`, `ANTHROPIC_API_KEY`, or the AWS variables.
- The env file is rewritten whole, not appended to, so a droplet switched from one mode to the other keeps none of the previous mode's variables. This is the assertion that makes a switch a switch.
- The script carries forward by exclusion, not by a list of names: every variable in the existing file survives a switch except the ones the mode itself decides. A list keeps only what somebody thought of, and a real sandbox carried `ANTHROPIC_MODEL`, which no such list held.
- `ANTHROPIC_MODEL` is one the mode decides, even though it is not a credential, because its value belongs to one mode's API. A Bedrock sandbox pins an inference profile such as `us.anthropic.claude-sonnet-5`, and Claude Code rejects that name under a subscription, failing every turn. A switch therefore drops it, reports the value it dropped, and writes a pin the operator supplies for the new mode instead — so the pin can move in the same step rather than vanish unnoticed.
- The script reads the existing file before it replaces it, and carries forward every value the new mode does not decide: the agent token, the control host, the GitHub token, the workspace, and the agent name. Rewriting whole is what makes a switch clean; reading first is what stops it from also being a wipe. The agent token is the one that cannot be recovered, because the control host stores only its hash.
- A credential prompt does not accept an empty answer. On a switch the old file has already been read by then, so an empty value would write an empty credential and leave the sandbox unable to authenticate at all.
- The script restarts the agent when one is running. systemd reads `EnvironmentFile` only at start, and a running agent hands its own environment to every Claude child, so without this a switch writes the new credential and the sandbox keeps billing the old one with nothing to show that it did not take.
- The script removes `/home/agent/.bashrc.d/spekk.sh` if it exists. An older version of the script exported `ANTHROPIC_API_KEY` there, and a droplet credentialed before then still carries a live key in every agent login shell. It is also in `agentSecrets`, so `spekk sandbox destroy` removes it from a machine spekk does not own.
- The credential is written to `/etc/spekk/agent.env` (mode 600) only, and the script sets `umask 077` before creating anything, so no file holding a secret exists even briefly at the default mode. Neither mode writes a model credential into `/home/agent/.bashrc.d/spekk.sh`; that file's `ANTHROPIC_API_KEY` export is removed. The systemd unit's `EnvironmentFile` is what the agent reads, so the shell-profile copy is a second place for a secret to leak from and buys nothing.
- The usage header documents an invocation that actually works, and no longer shows the `VAR=x ssh host 'bash -s' < script` form, which forwards nothing and whose prompts would eat the script from stdin. It tells the operator to pass non-secrets that way and to feed secrets from a root-only file instead, because a secret on a command line reaches the local shell history and both machines' process lists.
- `bash -n infrastructure/sandbox/setup-credentials.sh` parses clean, and the script keeps `set -euo pipefail`. (Not shellcheck: it is absent from this toolchain and from CI, which runs gofmt, build, `go test ./...`, and `spekk validate`. Adding a lint gate the project does not run belongs in its own change.)

**Note:** the prompts must not echo. A token typed at a visible prompt lands in the operator's terminal scrollback, which is the same exposure the env-file mode bits are there to prevent. Use `read -rs` for every credential.

## Known limit

The write path is driven end to end by a test, so a version that replaces the file without reading it first fails. `main`'s single call to that path is not covered: reaching `main` needs stubs for `cloud-init`, `systemctl`, and `su`, which is a large harness for one line.

## Verification

- Render the env file for both modes with a stub that runs the script's write step against a temporary path, and assert the present and absent variables per mode.
- Switch direction is covered explicitly: seed a file holding the Bedrock block, run the subscription path over it, and assert no Bedrock variable survives.
