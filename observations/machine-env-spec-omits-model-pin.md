---
slug: machine-env-spec-omits-model-pin
type: outdated_specs
severity: medium
status: open
created: 2026-10-30T12:00:00Z
announced: 2026-10-30T19:02:34Z
affected:
  - specs/sandbox-machine-mode/assertions/machine-model-credential.md
  - internal/sandbox/machine.go
  - internal/sandbox/auth.go
  - specs/sandbox-auth-mode/assertions/env-file-matches-auth-mode.md
---

# Machine-mode env file assertion predates the model pin

## Issue Description

Commit 896c7b8 (#326) made every new sandbox env file carry a default
`ANTHROPIC_MODEL` pin, including machine-mode sandboxes. It updated
`sandbox-auth-mode/env-file-matches-auth-mode`, but not
`sandbox-machine-mode/machine-model-credential`. That assertion is still
`done` and still says `anthropic_api_key` writes `ANTHROPIC_API_KEY` *only*.
The code now writes the key and a model pin. Two `done` assertions now
disagree about what goes in the same file.

## Evidence

- `specs/sandbox-machine-mode/assertions/machine-model-credential.md`, Success
  Criteria: "`anthropic_api_key` writes `ANTHROPIC_API_KEY` only." The
  `bedrock` and `none` bullets don't mention a pin either.
- `internal/sandbox/machine.go:513` — `buildMachineEnvContent` appends
  `modelLines(defaultModel(in.auth))` right after the credential lines.
- `internal/sandbox/auth.go:90` — `defaultModel` returns `claude-opus-5-5` for
  `AuthAPIKey`, `us.anthropic.claude-opus-5-5` for `AuthBedrock`, and `""` for
  `none`.
- `specs/sandbox-auth-mode/assertions/env-file-matches-auth-mode.md` was
  updated in the same commit: "Machine mode writes the same default for
  `bedrock`, `claude-opus-5-5` for `anthropic_api_key`, and no pin for
  `none`." `docs/cli-reference.md` (machine input table) says the same.

## Impact

A builder re-verifying `machine-model-credential` against its own text could
read the pin as a stray line and remove it, which would silently break the
1.50.0 model pin for machine-mode sandboxes. A reader of the machine-mode spec
gets a wrong picture of the env file.

## Recommendation

Amend `machine-model-credential` so the `anthropic_api_key` and `bedrock`
bullets allow the model pin, and `none` writes none. Point to
`env-file-matches-auth-mode` as the source of the pin values. The remedy
commit on this branch makes that edit.