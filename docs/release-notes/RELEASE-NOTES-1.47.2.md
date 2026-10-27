# Spekk CLI 1.47.2 - Agent ID Kept on Script-Equipped Hosts

This patch release makes `setup-credentials.sh` keep `SPEKK_AGENT_ID` in the agent settings file, and makes the CLI check the agent ID that the control host returns. There are no new commands or flags.

## The credential script keeps the agent ID

`infrastructure/sandbox/setup-credentials.sh` now writes `SPEKK_AGENT_ID` to `/etc/spekk/agent.env`. Before this release, the script ignored a `SPEKK_AGENT_ID` that you supplied, so a host that the script equipped had no agent ID, and each chat turn that selects a repository failed with "agent or session ID is missing".

- When you supply `SPEKK_AGENT_ID` in the environment, the script writes that value.
- When you do not supply it, the script keeps the value from the existing file.
- When the value is not a UUID, the script prints `ERROR: SPEKK_AGENT_ID must be a UUID` and exits with status 1. It does not change the file.

## Registration refuses an agent ID that is not a UUID

The CLI now refuses a registration response whose agent ID is not a UUID, before the ID can reach a host or a sandbox record. This applies to `POST /api/agents/`, which `create`, `provision`, and `register` use with `SPEKK_API_TOKEN`, and to `POST /api/agents/register/`, which `spekk sandbox create --machine` uses with a registration token. The error is "the response has an invalid agent id".

**Upgrade effect.** If an existing `agent.env` has a `SPEKK_AGENT_ID` line with a value that is not a UUID, the new script exits with status 1 each time it runs. Correct the line in `/etc/spekk/agent.env`, or run the script with a valid `SPEKK_AGENT_ID=<uuid>` in the environment. A file with no `SPEKK_AGENT_ID` line is not affected. Installing the new binaries alone does not change the file.
