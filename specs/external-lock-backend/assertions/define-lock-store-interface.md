---
id: define-lock-store-interface
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 1
status: not_started
branch: feature/external-lock-backend
---

# Define LockStore Interface

## What Must Be True

A Go interface `LockStore` exists in `internal/locks/` that defines the complete contract for external lock backends. All adapters (local, file, redis) implement this interface.

## Success Criteria

- ✅ New package `internal/locks/` exists with `lockstore.go`
- ✅ `LockStore` interface defined with exactly five methods:
  - `Acquire(ctx context.Context, assertionID string, info LockInfo) error`
  - `Release(ctx context.Context, assertionID, lockID string) error`
  - `Inspect(ctx context.Context, assertionID string) (*LockInfo, error)`
  - `List(ctx context.Context) ([]LockInfo, error)`
  - `ForceUnlock(ctx context.Context, assertionID string) error`
- ✅ `LockInfo` struct defined with: `ID`, `AssertionID`, `Holder`, `Hostname`, `PID`, `Branch`, `Created` (time.Time), `TTL` (time.Duration)
- ✅ Typed errors defined and exported:
  - `ErrLockHeld` — returned by `Acquire` when another holder has the lock
  - `ErrLockNotFound` — returned by `Inspect`/`Release` when no lock exists
  - `ErrLockExpired` — returned by `Release` when the lock TTL passed before release
- ✅ `ErrLockHeld` includes the current `LockInfo` of the holder so callers can display rich conflict errors
- ✅ Package doc comment on `lockstore.go` explains the contract and invariants
- ✅ Unit tests for `LockInfo` construction helpers (if any) and error wrapping

## Contract Invariants

Any implementation must satisfy:

1. `Acquire` is atomic — two concurrent callers for the same `assertionID` cannot both succeed
2. `Release` is idempotent — calling with a stale or already-released `lockID` returns `ErrLockNotFound` (not a panic or silent success with side effects)
3. `Acquire` with an expired lock of the same `assertionID` succeeds (stale recovery via TTL)
4. `ForceUnlock` always succeeds regardless of caller identity or lock state
5. All methods respect `context.Context` cancellation

## Out of Scope for This Assertion

- Actual backend implementations (separate assertions)
- Config loading (separate assertion)
- Wiring into `spekk next` (separate assertion)

## Notes

This is the contract-first step — no behavior changes anywhere else. The interface lets subsequent assertions build adapters and wire into the CLI in parallel.
