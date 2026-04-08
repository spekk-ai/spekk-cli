---
id: extract-local-backend
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 1
status: not_started
depends-on: define-lock-store-interface
branch: feature/external-lock-backend
---

# Extract Local Backend

## What Must Be True

The current git-native locking logic (reading `locked-by` from frontmatter, staleness check, `spekk next` filtering) is refactored into a `local` adapter that implements `LockStore`. Behavior is unchanged — this is a pure refactor.

## Success Criteria

- ✅ New file `internal/locks/local.go` contains `LocalBackend` struct implementing `LockStore`
- ✅ `LocalBackend.Acquire` writes the `locked-by` string to the assertion's frontmatter (same format as today: `builder-{hostname}-{pid}-{timestamp}`)
- ✅ `LocalBackend.Release` removes the `locked-by` field from frontmatter
- ✅ `LocalBackend.Inspect` reads the current `locked-by` value from the assertion file
- ✅ `LocalBackend.List` scans all assertion files and returns active (non-stale) locks
- ✅ `LocalBackend.ForceUnlock` removes `locked-by` from the assertion file regardless of holder
- ✅ Existing `IsLockStale` logic in `internal/parser/parser.go` is reused (or moved into `internal/locks/local.go` and the parser calls into it)
- ✅ `LocalBackend` uses the existing 2-hour staleness threshold by default
- ✅ All existing parser and builder tests pass unchanged
- ✅ New unit tests for `LocalBackend` covering: acquire, release, inspect, list, force-unlock, staleness, concurrent acquire on the same file (expect one failure)

## Refactor Constraints

- **No behavior change** — anything that uses `spekk next` today produces identical output after this assertion
- The parser's `FindNextAssertion` function still works against frontmatter for now; wiring into the `LockStore` interface happens in a later assertion
- `LocalBackend` is not yet constructed from config — that arrives with the config loader assertion

## Out of Scope

- Reading config to decide which backend to instantiate (separate assertion)
- Replacing the parser's inline lock check with a `LockStore` call (separate assertion — `wire-backend-into-next`)

## Notes

This assertion exists to give us a stable, test-covered reference implementation of the `LockStore` interface before we build `file` and `redis` adapters. If `LocalBackend` works correctly, the contract is validated; the other adapters just need to match it.
