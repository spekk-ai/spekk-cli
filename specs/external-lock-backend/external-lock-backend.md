---
id: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 1
---

# External Lock Backend

## Overview

Replace the current git-native assertion locking mechanism with a pluggable lock backend that supports external coordination stores. This enables real-time parallel work across multiple machines and teams without git race conditions, while preserving the full portability and standalone nature of spec files.

## Motivation

Today, assertion locking lives inside the spec file itself via a `locked-by` YAML field in frontmatter. This works for solo builders but breaks down for parallel work:

- **Race conditions**: Two builders can both run `spekk next`, both pick the same assertion, and both commit a `locked-by` field to the same file. The second push hits a non-fast-forward and the builder has to resolve a YAML merge conflict to recover.
- **Git history pollution**: Every lock/unlock creates a commit on a tracked file. Lock churn dominates the history of long-running specs.
- **No CLI surface**: There is no `spekk locks`, `spekk lock-info`, or `spekk force-unlock`. Recovery requires hand-editing YAML.
- **Opaque lock info**: The current `locked-by: builder-{hostname}-{pid}-{ts}` string doesn't capture branch, operation, or who the human is.
- **Stale lock handling is only time-based**: A crashed builder freezes work for up to 2 hours regardless of whether the process is actually alive.

## Design

### Separation of concerns

The critical insight: **locks and status are different kinds of state**.

| | Lifetime | Needs | Belongs in |
|---|---|---|---|
| **Status** | Durable (audit trail, review history) | Portable, reviewable, diffable | Git / frontmatter |
| **Locks** | Ephemeral (minutes–hours) | Atomic acquire, TTL, fast | External store |

Status stays exactly where it is today — in frontmatter, committed to git, self-contained, portable. Locks move out to a configurable external store.

**Portability is preserved.** Cloning the repo and reading a `.md` file still tells you everything about the assertion and its current status. Locks were never meaningful portable data, so externalizing them costs nothing.

### Architecture

```
┌────────────────────────┐         ┌────────────────────────┐
│ Git (source of truth)  │         │ Lock store (ephemeral) │
│                        │         │                        │
│ specs/**.md            │         │ active locks only      │
│   - declarative fields │         │   - assertion-id       │
│   - status (durable)   │         │   - holder info        │
│                        │         │   - TTL                │
│ PORTABLE               │         │ adapters:              │
│ SELF-CONTAINED         │         │   local, file, redis   │
└────────────────────────┘         └────────────────────────┘
```

### Backend adapters (in scope)

| Adapter | Mechanism | Use case |
|---|---|---|
| `local` (default) | Current git frontmatter `locked-by` field | Solo builders, zero config, unchanged behavior |
| `file` | `flock()` over a directory of lock files | Teams sharing a filesystem (NFS, shared drive, bind mount) |
| `redis` | `SET key value NX PX ttl` with Lua release | Teams with a Redis instance (self-hosted or managed) |

DynamoDB, S3, and other cloud blob stores are explicitly out of scope. Teams that want cloud coordination can run Redis on any host they control.

### Configuration

New file at repo root: `spekk.config.yaml`

```yaml
lock-backend:
  type: redis
  url: redis://coord.team.internal:6379
  ttl: 2h
```

Or file-based:

```yaml
lock-backend:
  type: file
  path: /Volumes/team-share/spekk-locks/
  ttl: 2h
```

Or explicit local (same as no config):

```yaml
lock-backend:
  type: local
```

**No config file → `local` backend → current behavior, fully backwards compatible.**

### Go interface

```go
// internal/locks/lockstore.go
type LockStore interface {
    Acquire(ctx context.Context, assertionID string, info LockInfo) error
    Release(ctx context.Context, assertionID, lockID string) error
    Inspect(ctx context.Context, assertionID string) (*LockInfo, error)
    List(ctx context.Context) ([]LockInfo, error)
    ForceUnlock(ctx context.Context, assertionID string) error
}

type LockInfo struct {
    ID          string        // unique lock identifier
    AssertionID string        // what's being locked
    Holder      string        // human-readable holder (user@host)
    Hostname    string
    PID         int
    Branch      string        // git branch of the holder
    Created     time.Time
    TTL         time.Duration
}
```

Typed errors: `ErrLockHeld` (someone else holds it), `ErrLockNotFound`, `ErrLockExpired`.

### CLI surface (new)

```bash
spekk locks                          # list all active locks (table or --json)
spekk lock-info <assertion-id>       # detail on a single lock
spekk force-unlock <assertion-id>    # remove lock regardless of holder
```

### Builder workflow changes

With a non-local backend configured, the builder:

1. Calls `LockStore.Acquire(assertionID, info)` before starting work
2. On `ErrLockHeld`, picks a different assertion (no YAML merge conflict path)
3. Calls `LockStore.Release(assertionID, lockID)` when marking `done`/`failed`
4. Does **not** write `locked-by` to frontmatter (status-only commits)

With the `local` backend, the builder behavior is unchanged — `locked-by` is still written to frontmatter exactly as today.

## Success Criteria

- ✅ Teams can run a shared Redis and have multiple builders claim work atomically with zero git races
- ✅ Solo builders see zero behavior change and no new ceremony (no config file required)
- ✅ Spec files remain fully portable — clone, open, read status, no backend required
- ✅ `spekk locks` shows who is working on what across the team in real time
- ✅ `spekk force-unlock` provides a recovery path when a builder crashes
- ✅ Stale locks are recovered via TTL regardless of backend
- ✅ All existing tests continue to pass (backwards compatibility)

## Out of Scope

- Externalizing **status** (status stays in git/frontmatter, non-negotiable for portability)
- Cloud backends (DynamoDB, S3, GCS, Azure) — teams can run Redis anywhere
- A central `spekk serve` coordinator for locks — the whole point is passive storage
- Offline/online reconciliation — if the backend is unreachable, `spekk next` fails fast with a clear error
- Lock history/audit trail in the backend — locks are ephemeral by design
- Migration of in-flight `locked-by` frontmatter values to an external store (just wait for current locks to drain)

## Assertions

See `assertions/` subfolder for detailed implementation criteria. Work on this spec lives on branch `feature/external-lock-backend`.
