---
id: spekk-toml-rejects-unknown-keys
parent: strictness-config
created: 2026-11-02T21:30:00Z
priority: 2
status: done
branch: feat/v2-strictness-config
depends-on: verify-runs-configured-commands
---

# A `spekk.toml` With an Unknown Key or Value Is Never Read as Valid

## Description

A typo such as `[strict.min-levels]` or `invarient = "property"` must fail loudly. If spekk skips it, the project believes that it has strictness when it has none.

## Success criteria

- The known tables are `[[verify.run]]` (keys `command` and `junit`), `[strict]` (key `unchecked`) and `[strict.min-level]` (one key for each property kind).
- An unknown table, an unknown key, a `min-level` key that is not a property kind, a level that is not on the ladder, or an `unchecked` value other than `warn` or `error` is invalid. A file that is not valid TOML is invalid.
- `spekk validate` fails on an invalid `spekk.toml` with one line `spekk.toml: <problem>` for each problem, and exits 1.
- `spekk verify` with an invalid `spekk.toml` prints the same problems, exits 1, and runs no command.
- A project with no `spekk.toml` validates as before.

**Tests:** `cmd/spekk/strictness_config_test.go`.
