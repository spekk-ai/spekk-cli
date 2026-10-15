---
id: destroy-revokes-first
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
depends-on: control-host-client
---

# `destroy` Revokes the Agent Before It Touches the Machine

**Tests:** internal/sandbox/register_test.go

## Success Criteria

- When the record has an agent id, `destroy` sends `POST /api/agents/<id>/revoke/` after the confirmation and before it stops the agent or deletes the machine.
- A failed revoke stops `destroy`: the machine, the key pair, and the record stay. The error names the agent id and says that `--ignore-revoke-failure` destroys the sandbox anyway.
- When the record has an agent id and `SPEKK_API_TOKEN` is empty, `destroy` stops before the confirmation and says that it cannot revoke the agent without `SPEKK_API_TOKEN`.
- `--force` only skips the confirmation. It does not continue after a failed revoke or a missing `SPEKK_API_TOKEN`.
- With `--ignore-revoke-failure`, a failed revoke or a missing `SPEKK_API_TOKEN` is a warning that names the agent id, and `destroy` continues.
- A record with no agent id makes no call to the control host, as before.

## Verification

- `go test ./internal/sandbox` covers a revoke that succeeds, a revoke that fails with `--force` (the record stays and the provider is not called), the same failure with `--ignore-revoke-failure`, and a missing `SPEKK_API_TOKEN` with and without `--ignore-revoke-failure`.
