---
id: script-equipped-host-keeps-agent-id
parent: agent-repository-preparation
created: 2026-10-23T22:40:00Z
priority: 1
status: done
branch: fix/script-agent-id
depends-on: every-host-knows-its-agent-id
---

# A script equipped host keeps its agent ID

**Tests:** internal/sandbox/setup_credentials_test.go, internal/sandbox/controlhost_test.go

The agent client needs its agent ID to get repository access. The credential script must write a valid ID and keep it on a later run.

## Success Criteria

- `setup-credentials.sh` writes a supplied `SPEKK_AGENT_ID` to `agent.env`, replaces an older ID when supplied, and keeps the existing ID when none is supplied.
- The script refuses an ID that is not a UUID before it replaces `agent.env`. An invalid supplied ID does not fall back to an existing ID.
- Both `agent.env.example` templates have a commented `SPEKK_AGENT_ID=` line and say that the ID comes from agent registration on the control host.
- `registerAgent` and `registerWithToken` refuse an agent ID that does not match `uuidRe` before the ID can reach `agent.env` or a sandbox record.

## Verification

- Run the script write path with a new ID, a prior ID, a replacement ID, and an invalid ID. Confirm the output file or the refusal in each case.
- Answer both registration routes with an invalid ID and confirm that each call fails.
