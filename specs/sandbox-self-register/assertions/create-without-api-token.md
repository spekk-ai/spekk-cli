---
id: create-without-api-token
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
depends-on: create-registers-agent
---

# `create` Without an API Token Keeps the Manual Path

A user who has no API token can still create a sandbox, as before.

## Success Criteria

- When `SPEKK_API_TOKEN` is empty, `create` makes no call to the control host. It makes the agent token itself and prints the registration details, as it did before this spec.
- It also prints how to register later: `spekk sandbox register <name>` with `SPEKK_API_TOKEN` set.
- `SPEKK_API_TOKEN` is not in the list of variables that `create` requires.

## Verification

- `go test ./internal/sandbox` runs `create` with no API token and an `httptest` control host, and checks that the server got no request and that the output names `spekk sandbox register`.
