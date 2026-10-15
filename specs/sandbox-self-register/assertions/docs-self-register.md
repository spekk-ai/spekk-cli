---
id: docs-self-register
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
depends-on: no-cloud-credential-to-control-host
---

# The Docs Describe Self-Registration and Schedules

## Success Criteria

- `docs/cli-reference.md` describes the registration step of `create`, the revoke step of `destroy` and its `--ignore-revoke-failure` flag, `spekk sandbox register`, the agent token rule of `provision`, and `spekk schedule` with each subcommand and flag.
- `docs/configuration.md` lists `SPEKK_API_TOKEN`, says which commands use it, and states the security rule: the control host receives only the API token and the agent token.
- `docs/advanced/sandbox-architecture.md` no longer says that registration is always a manual step.
- No doc names a private host as the control host.

## Verification

- Read the three files.
