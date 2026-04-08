---
id: spekk-locks-command
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 2
status: not_started
depends-on: wire-backend-into-next
branch: feature/external-lock-backend
---

# `spekk locks` and `spekk lock-info` Commands

## What Must Be True

Two new CLI subcommands expose the lock backend to users for visibility: `spekk locks` lists all active locks in a table, and `spekk lock-info <assertion-id>` shows detail for a single lock.

## Success Criteria

- ✅ `spekk locks` subcommand registered in the CLI
- ✅ Output (default): a table with columns `ASSERTION | HOLDER | BRANCH | AGE | TTL REMAINING`
- ✅ `spekk locks --json` outputs a JSON array of `LockInfo` objects
- ✅ Empty state: prints `No active locks.` (not an error)
- ✅ Results sorted by `Created` ascending (oldest first)
- ✅ `spekk lock-info <assertion-id>` subcommand registered
- ✅ Output shows full `LockInfo` as a key/value block:
  ```
  assertion:  dashboard-loads-fast
  holder:     alice@macbook
  hostname:   macbook
  pid:        12345
  branch:     feature/dashboard
  lock id:    builder-macbook-12345-1706210400
  created:    2026-04-08 15:37:12
  age:        23m
  ttl:        2h
  expires in: 1h37m
  ```
- ✅ `spekk lock-info --json <assertion-id>` outputs single `LockInfo` as JSON
- ✅ `spekk lock-info` on a non-existent lock returns exit code 1 with error `no active lock for assertion: {id}`
- ✅ Both commands work against any configured backend (`local`, `file`, `redis`)
- ✅ Both commands fail with the same clear backend-unreachable error as `spekk next` if the backend is down
- ✅ Integration tests for each backend type verify command output format
- ✅ Both commands documented in `spekk --help` output

## Implementation Notes

- Use an existing table renderer in the codebase if one exists (check `internal/show/`); otherwise implement a minimal aligned formatter
- `AGE` and `TTL REMAINING` are computed at print time from `LockInfo.Created` and `LockInfo.TTL`
- Use human-readable durations (e.g., `23m`, `1h12m`, `2d`) not raw seconds

## Out of Scope

- `--watch` live updates (can reuse `spekk show --watch` infrastructure later)
- Filtering by holder/branch/age (future ergonomics)
- Deleting locks (that's `spekk force-unlock`, separate assertion)

## Notes

This is the "I can finally see what's happening" assertion. Today there's no way to ask "what's locked?" without grepping spec files. With this, the team has real-time visibility.
