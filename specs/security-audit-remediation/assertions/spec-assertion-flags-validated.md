---
id: spec-assertion-flags-validated
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 1
status: not_started
branch: feature/spekk-sandbox-vulnrabilities
---

# Spec and assertion CLI flags are validated before prompt interpolation

The `--spec` and `--assertion` flags in the builder command are validated to contain only valid spec/assertion ID characters before being interpolated into Claude's activation message. Arbitrary text in these flags cannot inject instructions into the prompt.

## Success Criteria

- `cfg.Spec` and `cfg.Assertion` values are validated against `^[a-z0-9-]+$` before use in `BuildActivationMessage` or prompt string construction
- Validation occurs before the values reach `BuildSpekkNextCommand` or the `fmt.Sprintf` at line 440
- Invalid values (containing spaces, quotes, newlines, markdown, or instruction-like text) are rejected with a clear error message
- Valid kebab-case IDs (`my-spec`, `fix-login-bug`) are accepted
- Tests cover accepted and rejected flag values
