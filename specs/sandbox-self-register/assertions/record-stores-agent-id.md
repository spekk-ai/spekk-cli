---
id: record-stores-agent-id
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
---

# The Sandbox Record Stores the Agent Id and No Credential

`~/.config/spekk/sandboxes.json` records which agent on the control host belongs to each sandbox, so `destroy`, `provision`, and `spekk schedule` can find it.

## Success Criteria

- `SandboxMeta` has an `AgentID` field, stored as `agentId`, and omitted when it is empty. A record written before the field existed reads back with an empty `AgentID`.
- `SandboxMeta` has no field for the API token, the agent token, or any cloud or model credential.

## Verification

- `go test ./internal/sandbox` saves a record with an agent id, reads the file, and checks that it has the id and none of the token values used in the test.
