---
slug: auth-docs-assertion-says-no-path-writes-api-key
type: outdated_specs
severity: medium
status: open
created: 2026-11-14T00:00:00Z
affected:
  - specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md
  - internal/sandbox/machine.go
  - internal/sandbox/auth.go
  - docs/cli-reference.md
---

# Auth-docs assertion says no provisioning path writes ANTHROPIC_API_KEY

## Issue Description

The `auth-modes-are-documented` assertion (status `done`) justifies removing
the `ANTHROPIC_API_KEY` line from `agent.env.example` with "after this change
no provisioning path writes that variable", and its Verification step expects
`grep -rn ANTHROPIC_API_KEY docs/ internal/sandbox/ infrastructure/` to return
only test lines because "no shipped file writes it". Machine mode added a
provisioning path that does write it: the `anthropic_api_key` model credential.

## Evidence

- `internal/sandbox/machine.go:190-201`: `case "anthropic_api_key"` returns
  `AuthAPIKey` with `map[string]string{apiKeyVar: c.APIKey}`.
- `internal/sandbox/auth.go:26` defines `apiKeyVar = "ANTHROPIC_API_KEY"`;
  `auth.go:71` emits it as the env-file line for `AuthAPIKey`.
- `docs/cli-reference.md:1085` documents it: "`anthropic_api_key` |
  `api_key` | `ANTHROPIC_API_KEY`, and the model pin ...".
- Running the assertion's Verification grep returns non-test hits, among them
  `docs/cli-reference.md:1085`, `docs/advanced/sandbox-architecture.md:188`,
  `internal/sandbox/auth.go:26` and `internal/sandbox/machine.go` (through
  `apiKeyVar`).

## Impact

A verifier running the Verification step sees it fail against correct code.
Worse, a builder could read the assertion as a requirement to remove the
machine-mode API-key credential. What the assertion really means is narrower:
the terminal `--auth` modes (`bedrock`, `subscription`) never write the
variable, and the old `.bashrc.d` export is gone.

## Recommendation

Scope the claim and the grep to the terminal `--auth` modes, and name machine
mode's `anthropic_api_key` credential as the one path that writes the
variable on purpose. The remedy commit on this branch proposes that wording.
