---
id: file-lock-backend
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 2
status: not_started
depends-on: define-lock-store-interface
branch: feature/external-lock-backend
---

# File Lock Backend

## What Must Be True

A `file` backend adapter implements `LockStore` using OS-level `flock()` on per-assertion lock files in a configured directory. This enables teams sharing a filesystem (NFS, SMB, bind mount, shared drive) to coordinate locks without any additional infrastructure.

## Success Criteria

- ✅ New file `internal/locks/file.go` contains `FileBackend` struct implementing `LockStore`
- ✅ Constructor `NewFileBackend(dir string, defaultTTL time.Duration) (*FileBackend, error)` creates the lock directory if missing
- ✅ `Acquire`:
  - Creates `{dir}/{assertionID}.lock` using `O_CREATE|O_EXCL|O_RDWR`
  - Takes an exclusive `flock` on the file
  - Writes the `LockInfo` as YAML content inside the file
  - Returns `ErrLockHeld` (with current holder info) if the file already exists and is not expired
  - Succeeds by overwriting if the existing lock is past its TTL (stale recovery)
- ✅ `Release`:
  - Reads the lock file, verifies `lockID` matches
  - Deletes the file under an `flock`
  - Returns `ErrLockNotFound` if the file does not exist
- ✅ `Inspect`: reads and parses the lock file; returns `ErrLockNotFound` if missing
- ✅ `List`: reads all `*.lock` files in the directory, parses them, filters out expired ones
- ✅ `ForceUnlock`: deletes the lock file regardless of holder
- ✅ TTL is enforced by comparing `LockInfo.Created + LockInfo.TTL` against `time.Now()` on read
- ✅ Lock files use YAML format for human-readability:
  ```yaml
  id: builder-macbook-12345-1706210400
  assertion-id: dashboard-loads-fast
  holder: alice@macbook
  hostname: macbook
  pid: 12345
  branch: feature/dashboard
  created: 2026-04-08T16:00:00Z
  ttl: 2h
  ```
- ✅ Unit tests cover: acquire, double acquire (second fails), release, stale acquire succeeds, inspect missing, list multiple, force-unlock
- ✅ Concurrency test: two goroutines racing to acquire the same assertion — exactly one succeeds
- ✅ NFS/flock caveat documented in code comments (flock over NFSv3 can be unreliable; works on local FS, NFSv4, SMB, most shared drives)

## Implementation Notes

- Use `golang.org/x/sys/unix.Flock` for portability across macOS and Linux (Windows uses `LockFileEx`, but for MVP Unix-only is acceptable — note in doc comments)
- Lock file format intentionally matches `LockInfo` 1:1 so humans can `cat` a stuck lock file to debug

## Out of Scope

- Windows support (can be added later with a separate build tag)
- Retries with backoff (caller handles conflicts by picking another assertion)
- Lock hierarchies / nested locks

## Notes

The `file` backend is specifically designed for "we already share a filesystem" teams — think a monorepo on a shared dev VM, or a mounted NAS. No additional services, no credentials, just a directory both machines can see.
