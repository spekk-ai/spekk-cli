---
id: no-cloud-credential-to-control-host
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
depends-on: schedule-commands
---

# No Request to the Control Host Carries a Cloud or Model Credential

The control host never receives, stores, or logs a cloud credential or a model credential of the user. These credentials go from the user's machine to the user's sandbox only.

## Success Criteria

- No request from the CLI to the control host carries the value of `DO_API_TOKEN`, `DIGITALOCEAN_TOKEN`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `GOOGLE_APPLICATION_CREDENTIALS`, `CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, or `GITHUB_TOKEN`, in its URL, its headers, or its body.
- The only credential in a request is the `Authorization: Token <SPEKK_API_TOKEN>` header.
- No flag, field, or request body of the control host client takes a cloud or model credential.

## Verification

- A test in `internal/sandbox` sets each variable above to a unique value, then runs `create`, `register`, `provision`, `destroy`, and every `spekk schedule` subcommand against one `httptest` control host that records each request. It fails if any recorded URL, header, or body contains one of the values, or if a request has an `Authorization` header other than the API token.
