---
id: agent-reports-sign-in-required
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-machine-mode
depends-on: machine-model-credential
---

# An Agent With No Claude Credential Reports `sign_in_required`

A personal sandbox (`model_credential` `none`) has no model credential until the person signs in to Claude Code on the box, with Claude Code's own login. Before this spec, each turn on such a box started `claude`, which failed, and the person saw only an exit status.

**Tests:** cmd/sandbox/claude_auth_test.go

## Success Criteria

- The agent client has a model credential when its environment has a non-empty `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `CLAUDE_CODE_USE_BEDROCK`, `CLAUDE_CODE_USE_VERTEX`, or `CLAUDE_CODE_USE_FOUNDRY`, or when the Claude Code sign-in file exists (`$CLAUDE_CONFIG_DIR/.credentials.json`, else `$HOME/.claude/.credentials.json`). The client reads this again for each heartbeat and each turn, so a sign-in takes effect with no restart.
- Each heartbeat frame has a field `claude_auth`: `"ready"` when the client has a model credential, else `"sign_in_required"`. The state name for the server is `sign_in_required`.
- A turn on a client with no model credential does not start `claude`. The client answers with an error frame for that turn: "Claude Code is not signed in on this sandbox. Sign in on the sandbox with `claude` as the agent user, then send the message again."
- The field is additive, so `ProtocolVersion` goes from `1.1` to `1.2`. A control host that does not know the field ignores it.
- A file that exists but is not valid is not checked. Claude Code gives its own error for it, which is better than a guess.

## Verification

- One table test of the check: no credential, each variable, and the sign-in file under `HOME` and under `CLAUDE_CONFIG_DIR`.
- One test sends a turn to a client with no credential and checks the error frame and that no `claude` process started.
