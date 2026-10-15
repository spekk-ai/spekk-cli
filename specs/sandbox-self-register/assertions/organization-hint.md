---
id: organization-hint
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
depends-on: control-host-client
---

# A Refused Organization Tells the User How to Choose One

`POST /api/agents/` answers `400` with an `organization` key when the control host cannot choose the organization of the agent. The usual cause is a user who belongs to more than one organization and gave no `--organization`. The real body is `{"organization": ["You belong to several organizations. Specify one."]}`.

**Tests:** internal/sandbox/controlhost_test.go

## Success Criteria

- When the registration answer is `400` and its body has an `organization` key, the error keeps the text of the control host, and adds a hint: pass `--organization <id>`, and find the id in the `organization` field of an agent you already have (`GET /api/agents/`), or ask an admin of the organization.
- `create`, `provision`, and `register` all show this hint, because they register through the same function.
- A `400` without an `organization` key gets no hint.

## Verification

- `go test ./internal/sandbox` answers a registration with the real body above and checks the hint, and answers with a `sandbox_id` error and checks that there is no hint.
