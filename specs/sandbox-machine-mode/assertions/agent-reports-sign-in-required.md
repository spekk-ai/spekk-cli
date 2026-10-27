---
id: agent-reports-sign-in-required
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-machine-mode
depends-on: machine-model-credential
---

# An Agent With No Claude Credential Reports `claude_signed_in: false`

A personal sandbox (`model_credential` of kind `none`) has no model credential until the person signs in to Claude Code on the box, with Claude Code's own login. Before this spec, each turn on such a box started `claude`, which failed, and the person saw only an exit status.

**Tests:** cmd/sandbox/claude_auth_test.go

## Success Criteria

- The agent client has a model credential when its environment has a non-empty `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `CLAUDE_CODE_USE_BEDROCK`, `CLAUDE_CODE_USE_VERTEX`, or `CLAUDE_CODE_USE_FOUNDRY`, or when the Claude Code sign-in file (`$CLAUDE_CONFIG_DIR/.credentials.json`, else `$HOME/.claude/.credentials.json`) is a regular file that the agent can open and whose size is more than zero. The agent does not read its content (`sign-in-check-reads-no-credential`). The client reads this again for each heartbeat and each turn, so a sign-in takes effect with no restart.
- Each heartbeat frame has a field `claude_signed_in`, a JSON boolean: `true` when the client has a model credential, else `false`. The server shows "Sign in to Claude" while it is `false`.
- A turn on a client with no model credential does not start `claude`. The client answers with an error frame for that turn: "Claude Code is not signed in on this sandbox. Sign in on the sandbox with `claude` as the agent user, then send the message again."
- The field is additive, so `ProtocolVersion` goes from `1.1` to `1.2`. A control host that does not know the field ignores it.
- The CLI does not read or parse the sign-in file. When it is not valid, Claude Code gives its own error for it, which is better than a guess.

## Verification

- One table test of the check: no credential, one variable, the sign-in file under `HOME` and under `CLAUDE_CONFIG_DIR`, an empty sign-in file, and a sign-in file that the agent cannot read.
- One test sends a turn to a client with no credential and checks the error frame and that no `claude` process started.
