---
id: wire-backend-into-next
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 1
status: not_started
depends-on: extract-local-backend
branch: feature/external-lock-backend
---

# Wire Backend Into `spekk next`

## What Must Be True

`spekk next` uses the configured `LockStore` (from `spekk.config.yaml`) to filter locked assertions, replacing the inline `locked-by` frontmatter check in the parser. When the backend is `local`, behavior is identical to today. When the backend is `file` or `redis`, locks come from the external store.

## Success Criteria

- ✅ CLI entry point for `spekk next` loads `spekk.config.yaml` via the config loader
- ✅ Instantiates the correct `LockStore` adapter based on `config.LockBackend.Type`:
  - `local` → `LocalBackend` (current behavior)
  - `file` → `FileBackend` from configured path
  - `redis` → `RedisBackend` from configured URL
- ✅ `FindNextAssertion` (or its caller) accepts a `LockStore` and calls `LockStore.List()` to build the set of currently-locked assertions, replacing the inline `IsLockStale`/frontmatter check
- ✅ Locked assertions are filtered out of the candidate set exactly as before
- ✅ When `spekk next` returns nothing AND all active candidates were filtered by locks, the output includes a human-readable summary of who holds what (from `LockStore.List()`)
- ✅ If the configured backend is unreachable (e.g. Redis down, file path missing), `spekk next` fails with a clear error naming the backend, its config, and the underlying error — it does **not** silently fall back to `local`
- ✅ A new `--no-lock-check` flag bypasses lock filtering entirely (parallel to Terraform's `-lock=false` escape hatch)
- ✅ All existing `spekk next` tests continue to pass when run without a config file (`local` default)
- ✅ New integration tests verify:
  - `file` backend blocks a locked assertion from being selected
  - `redis` backend blocks a locked assertion from being selected (using miniredis)
  - Unreachable backend produces a clear error

## Error Message Example

When the backend is unreachable:

```
Error: lock backend unreachable
  type:  redis
  url:   redis://coord.team.internal:6379
  cause: dial tcp 10.0.0.5:6379: connect: connection refused

Fix: verify the backend is running, or run with --no-lock-check to bypass.
```

When all candidates are locked:

```
No eligible assertions.

Currently locked (blocking work):
  dashboard-loads-fast    alice@macbook    feature/dashboard    23m ago
  export-to-csv           bob@workstation  feature/export       1h12m ago

Run `spekk locks` to see all active locks.
```

## Out of Scope

- `spekk locks` and `spekk force-unlock` CLI commands (separate assertions)
- Builder-side acquire/release through the backend (separate assertion)
- Watch/poll mode for live lock updates (future work)

## Notes

This is the assertion that makes the backend real from a user's perspective. Until this ships, everything else is scaffolding. After this ships, `spekk next` respects external locks.
