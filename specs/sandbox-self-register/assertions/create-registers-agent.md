---
id: create-registers-agent
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
depends-on: control-host-client
---

# `create` Registers the Agent Before It Provisions

When `SPEKK_API_TOKEN` is set, `spekk sandbox create` registers the agent on the control host before it creates or equips a machine. The agent token on the machine is the one the control host made.

**Tests:** internal/sandbox/register_test.go

## Success Criteria

- `create` sends `POST /api/agents/` after it fetches the release artifacts and before it asks the provider for a machine. The body has `name` (the sandbox name), `sandbox_id` (`spekk-<name>-<8 random hex characters>`), and `organization` only when `--organization <id>` is given.
- A failed registration stops `create` before it creates anything billable.
- The `auth_token` from the response is the `SPEKK_AGENT_TOKEN` that `create` writes to `/etc/spekk/agent.env`. `create` does not make a token of its own in this case.
- The record stores the agent `id` from the moment the record is first saved.
- When a later step fails (the provider, the wait, the equip steps, or the final save), `create` revokes the agent it registered and says so on stderr. When the revoke succeeds, the record no longer has the agent id. When the revoke fails, `create` prints the agent id and a warning, and the record keeps the id so that `destroy` can revoke it.
- When `create` gets `SIGINT` (Ctrl-C) or `SIGTERM` after the registration, it revokes the agent, removes the agent id from the record when a record exists, and then exits with status `130` for `SIGINT` or `143` for `SIGTERM`. A failed revoke prints the agent id, as above.
- On success, `create` prints the agent id. It does not print the agent token.

**Note:** a revoked agent keeps its `sandbox_id` on the control host, so the next registration of the same sandbox name must send a different `sandbox_id`.

## Verification

- `go test ./internal/sandbox` runs `create` for an existing machine against an `httptest` control host. One test checks the order of calls and the token that reaches the equip step. One test makes the equip step fail and checks that the server got a revoke for the same agent. One test makes the provider fail and checks the revoke. One test makes the revoke fail and checks that the record keeps the agent id. One test sends `SIGINT` to the process during the equip step and checks the revoke, the record, and the exit status.
