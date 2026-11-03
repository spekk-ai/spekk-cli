---
id: strictness-sets-the-ci-result
parent: strictness-config
created: 2026-11-02T21:30:00Z
priority: 2
status: done
branch: feat/v2-strictness-config
depends-on: evidence-level-is-strongest-passing
---

# `spekk verify` Fails Exactly When a `done` Property Breaks a Strictness Rule

## Description

After a run, `spekk verify` applies the `[strict]` rules to each `done` property. A broken rule is an error or a warning, as the configuration says.

## Success criteria

- With `unchecked = "error"`, each `done` property whose truth is `unchecked` or `unconfirmed` prints one line `strict: <id> is <truth>` and makes `spekk verify` exit 1.
- With `unchecked = "warn"` or no setting, the same line goes to stderr with the prefix `warning:`, and the exit code does not change.
- With `[strict.min-level] <kind> = "<level>"`, each `done` property of that kind whose level is below `<level>`, or empty, prints `strict: <id> is <level or no level>, below <level> for <kind>` and makes `spekk verify` exit 1.
- A property that is not `done` never breaks a strictness rule.
- A `failing` property makes `spekk verify` exit 1 with or without `[strict]` (`verify-command`).
- **Tests:** `unchecked = "error"` with one `done` unchecked property (exit 1) and one `not_started` unchecked property (no line); a minimum level met and not met; `warn` keeps exit 0.

**Test file:** `internal/verify/strictness_test.go`.
