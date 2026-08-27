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
- The credential is written to `/etc/spekk/agent.env` (mode 600) only. Neither mode writes a model credential into `/home/agent/.bashrc.d/spekk.sh`; that file's `ANTHROPIC_API_KEY` export is removed. The systemd unit's `EnvironmentFile` is what the agent reads, so the shell-profile copy is a second place for a secret to leak from and buys nothing.
- The usage header documents an invocation that actually works, and no longer shows the `VAR=x ssh host 'bash -s' < script` form, which forwards nothing and whose prompts would eat the script from stdin.
- `bash -n infrastructure/sandbox/setup-credentials.sh` parses clean, and the script keeps `set -euo pipefail`. (Not shellcheck: it is absent from this toolchain and from CI, which runs gofmt, build, `go test ./...`, and `spekk validate`. Adding a lint gate the project does not run belongs in its own change.)

**Note:** the prompts must not echo. A token typed at a visible prompt lands in the operator's terminal scrollback, which is the same exposure the env-file mode bits are there to prevent. Use `read -rs` for every credential.

## Verification

- Render the env file for both modes with a stub that runs the script's write step against a temporary path, and assert the present and absent variables per mode.
- Switch direction is covered explicitly: seed a file holding the Bedrock block, run the subscription path over it, and assert no Bedrock variable survives.
