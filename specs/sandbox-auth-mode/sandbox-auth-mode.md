---
id: sandbox-auth-mode
created: 2026-08-27T00:00:00Z
priority: 1
---

# A sandbox chooses how it pays for Claude

## Problem

Every sandbox that `spekk sandbox create` builds is hardwired to bill Claude usage through Bedrock. `buildEnvContent` (`internal/sandbox/commands.go:682`) writes `CLAUDE_CODE_USE_BEDROCK=1` into `/etc/spekk/agent.env` for every droplet, and `Create` (`internal/sandbox/commands.go:58`) refuses to start without `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_DEFAULT_REGION`. An operator who wants a sandbox to authenticate any other way has no way to ask for one.

Claude Code accepts a second credential: a long-lived subscription token, minted by `claude setup-token` and read from the `CLAUDE_CODE_OAUTH_TOKEN` environment variable. Both variable names ship in the Claude Code binary, so this is a supported path and not a workaround.

The change is small because the plumbing already exists. `cloud-init.yaml` gives the agent service `EnvironmentFile=/etc/spekk/agent.env`, and `headlessChildEnv` (`internal/agent/launcher_headless_env.go`) hands the process environment to the `claude -p` child unchanged (`internal/agent/launcher_headless_unix.go`). The env file is the only thing that decides which credential a sandbox uses. Nothing between the env file and the model call needs to change.

## The two provisioning paths disagree today

There are two ways to put credentials on a sandbox, and they write different files.

| | `buildEnvContent` (Go) | `setup-credentials.sh` |
|---|---|---|
| Model credential | AWS trio + `CLAUDE_CODE_USE_BEDROCK=1` | `ANTHROPIC_API_KEY` |
| `WORKSPACE`, `SPEKK_AGENT_NAME` | written | absent |

So a droplet built by the CLI bills through Bedrock, and the same droplet re-credentialed by the script bills through a direct API key. Neither path knows about the other. Adding a mode to only one of them would deepen the split, so both learn the same two modes and the script's Bedrock branch is brought into line with the Go path.

The script has a second defect that blocks the switching work this spec exists for. Its usage header (`infrastructure/sandbox/setup-credentials.sh:11`) documents `SPEKK_AGENT_TOKEN=xxx ... ssh root@IP 'bash -s' < setup-credentials.sh`. That cannot work in either half: the assignments apply to the local `ssh` process and SSH does not forward them, and the fallback `read -rp` prompts cannot run either, because the remote shell's stdin is the script itself, so each `read` consumes the next line of the script instead of waiting for an operator.

## Solution

`spekk sandbox create --auth <mode>` with two modes.

- `bedrock` — the default: the same variables with the same values the command writes today. An operator who does not pass the flag sees no change in behavior. The one difference is line order — `CLAUDE_CODE_USE_BEDROCK` moves up to sit with the AWS credentials, so each mode's block is contiguous and its absence is visible at a glance. A systemd `EnvironmentFile` gives line order no meaning.
- `subscription` — writes `CLAUDE_CODE_OAUTH_TOKEN` and omits every Bedrock variable.

The credential reaches the CLI the same way `GITHUB_TOKEN` and the AWS variables already do: from the operator's own environment, so it can come from a keychain per command and never has to sit in a file.

## Absence is the requirement, not just presence

This is the part an implementer can get wrong by doing half the job. `CLAUDE_CODE_USE_BEDROCK` and `ANTHROPIC_API_KEY` each select a credential path of their own. A file that sets the subscription token *and* leaves either of those in place is ambiguous, and the sandbox may keep billing exactly the way the operator was trying to stop. So the criterion for subscription mode is that the file **does not contain** `CLAUDE_CODE_USE_BEDROCK`, `ANTHROPIC_API_KEY`, or the AWS credentials — not merely that it also contains the token.

This matters most when switching a sandbox that already exists, which is the reason this spec is being built: the old variables are already on that droplet, and the switch has to remove them.

## Scope

In scope: the `--auth` flag, the env file the two provisioning paths write, the required-variable check, and the operator documentation that tells someone which variable to set.

Out of scope, deliberately:

- **A provider abstraction.** Claude Code also reads `CLAUDE_CODE_USE_VERTEX` and `CLAUDE_CODE_USE_FOUNDRY`. Nobody has asked for either. Two modes is a branch; four is a registry, and a registry earns its keep only once a third real caller exists.
- **A `spekk sandbox switch-auth` command.** Re-credentialing a live droplet is what `setup-credentials.sh` is for, and teaching it the mode costs one parameter. A new Go subcommand would duplicate the script.
- **Recording the mode in sandbox metadata.** Nothing reads it back. The env file on the droplet is the truth, and `spekk sandbox ssh <name> 'grep -c CLAUDE_CODE_USE_BEDROCK /etc/spekk/agent.env'` answers the question without a new field to keep in sync.
- **Validating the token's shape.** The credential is opaque to us. A bad one produces a real authentication error from the agent, which is a better message than any guess we could make from a prefix.
- **Reconciling the rest of the gap between the two provisioning paths.** Only the model-credential block is brought into line here. The `WORKSPACE` and `SPEKK_AGENT_NAME` difference is real but unrelated, and folding it in would hide an auth change inside a general cleanup.

## Assertions

1. `auth-flag-selects-mode` — `spekk sandbox create --auth` accepts the two modes and rejects anything else before it spends money
2. `env-file-matches-auth-mode` — each mode writes its own credential and none of the other's
3. `create-requires-its-modes-credentials` — the pre-flight check asks for what the chosen mode actually uses
4. `setup-credentials-switches-a-live-sandbox` — the script can move an existing droplet between modes, and its usage header describes an invocation that works
5. `auth-modes-are-documented` — an operator can find out which variable to set without reading Go
