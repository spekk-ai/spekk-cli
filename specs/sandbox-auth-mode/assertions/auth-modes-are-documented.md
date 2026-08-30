---
id: auth-modes-are-documented
parent: sandbox-auth-mode
created: 2026-08-27T00:00:00Z
priority: 2
status: done
branch: feat/sandbox-subscription-auth
depends-on: env-file-matches-auth-mode
---

# An Operator Can Learn Which Variable to Set Without Reading Go

The two auth modes, and the credential each one needs, are documented where an operator provisioning a sandbox will look: the configuration reference and the example env file the droplet ships with.

## Success Criteria

- `docs/configuration.md` states in its sandbox-provisioning section that `spekk sandbox create --auth` chooses the credential, names both modes, and says `bedrock` is the default.
- Its provisioning-variable table marks which variables belong to which mode, rather than listing all of them as unconditional. `DO_API_TOKEN`, `GITHUB_TOKEN`, and `SPEKK_HOST` are needed by both; the AWS trio is Bedrock only; `CLAUDE_CODE_OAUTH_TOKEN` is subscription only.
- The `CLAUDE_CODE_OAUTH_TOKEN` row says the value comes from `claude setup-token` and needs a Claude subscription, so a reader knows how to obtain one.
- Its agent-runtime table gains the model-credential variables the agent actually reads, marked as set by whichever mode provisioned the sandbox.
- The `agent.env.example` block in `internal/sandbox/cloud-init.yaml` shows both modes' credentials as alternatives, each with a comment saying which mode writes it. Its `ANTHROPIC_API_KEY` line is removed, because after this change no provisioning path writes that variable.
- The documentation says how to obtain the token when no browser is available at the terminal: the command prints a URL and waits for a code, the URL may be opened on any device, and the code is pasted back. It records the two facts that are invisible until they bite — a carriage return sent in the same write as the pasted code is swallowed by the paste handler and does not submit, and restarting the command invalidates any code already issued.
- The documentation says where to keep the token: a file only the operator can read, never a command line, because an inline secret reaches shell history and both machines' process lists.
- The documentation warns that a subscription seat's rate limit is shared across every session using it, so several sandboxes on one seat contend with each other and with the operator's own use. This is the fact that decides how many sandboxes belong on a subscription, and it is invisible from the flag alone.
- No real sandbox name, host, or account appears in any of it. This repository is public; the examples are placeholders.

## Verification

- Read the rendered tables and confirm every variable the code reads appears exactly once, in the right mode column.
- `grep -rn ANTHROPIC_API_KEY docs/ internal/sandbox/ infrastructure/` returns only test lines, which name the variable to assert its absence. No shipped file writes it, so it is gone from every provisioning path rather than only from the Go one.
