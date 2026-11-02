---
id: strictness-config
created: 2026-11-02T21:30:00Z
priority: 2
---

# Strictness in `spekk.toml` - a Project Makes Properties Stricter One Kind at a Time

## Problem

A v1 project has hundreds of `done` assertions and no evidence. It cannot make all of them strict at one time. As in TypeScript, the project turns on strictness where the value is high: one kind of property at a time, and one rule at a time.

## Configuration

```toml
[strict]
unchecked = "warn"        # warn | error

[strict.min-level]
invariant = "property"
authorization = "example"
```

## Decisions

- `spekk.toml` sets a minimum evidence level for each kind of property, and sets whether an unchecked property is `warn` or `error` in CI (William, 2026-11-01).
- Strictness only decides whether CI fails. It does not change what spekk displays.

## Open choices

- **The CI command.** `spekk verify` applies strictness, because it has the results. `spekk validate` only checks that `spekk.toml` is valid.
- **Done properties only.** Strictness applies to properties whose workflow value is `done`. A `not_started` property has no code yet, so it is correct that nothing checks it. Without this rule, each new spec fails CI.
- **Unconfirmed counts as unchecked.** `unchecked = "error"` also fails on an `unconfirmed` property. Both are prose only.
- **Minimum level is always an error.** A `done` property below the minimum level of its kind fails CI. There is no `warn` for it, because a project sets a minimum level only when it wants CI to enforce it.
- **Defaults.** With no `spekk.toml` or no `[strict]` table, `unchecked` is `warn` and there is no minimum level.
- **Unknown keys fail.** A typo in strictness must not turn strictness off with no message.
- **This repository.** The `.gitignore` of spekk-cli has `/spekk*`, which also ignores `/spekk.toml`. The dogfood task must add `!/spekk.toml`.

## Assertions

See `assertions/` for what must be true.
