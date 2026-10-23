---
id: setup-credentials-writes-agent-id
parent: agent-repository-preparation
created: 2026-10-23T18:00:00Z
priority: 1
status: not_started
depends-on: every-host-knows-its-agent-id
---

# `setup-credentials.sh` writes the agent ID like the other writers

**Tests:** internal/sandbox/setup_credentials_test.go

`every-host-knows-its-agent-id` made `create`, `provision`, and `register` write `SPEKK_AGENT_ID` to `/etc/spekk/agent.env`. `infrastructure/sandbox/setup-credentials.sh` is also a writer of that file, and its header says it equips a freshly provisioned droplet. It does not prompt for `SPEKK_AGENT_ID` or render it. A droplet that it equips from scratch fails each repository chat turn with "agent or session ID is missing". On an existing file, the line survives only because unknown keys pass through unchanged.

## Success Criteria

- `SPEKK_AGENT_ID` is one of the variables that the script renders itself (`KNOWN_VARS`), next to `SPEKK_AGENT_TOKEN`.
- On a run with no existing value, the script takes `SPEKK_AGENT_ID` from the environment, or asks for it with a plain prompt (it is not a secret), in the same way as `SPEKK_HOST`.
- On a mode switch or a same-mode re-run, an existing `SPEKK_AGENT_ID` is carried over unchanged, unless the caller supplies a new one.
- The script header and the setup-credentials section of `docs/configuration.md` say where the operator finds the agent ID.

## Verification

- A test renders the env file on a fresh run with `SPEKK_AGENT_ID` in the environment, and finds the line.
- A test switches an existing file between modes, and finds the original `SPEKK_AGENT_ID` line once in the result.
