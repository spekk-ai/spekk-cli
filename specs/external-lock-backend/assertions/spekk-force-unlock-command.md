---
id: spekk-force-unlock-command
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 2
status: not_started
depends-on: wire-backend-into-next
branch: feature/external-lock-backend
---

# `spekk force-unlock` Command

## What Must Be True

A new `spekk force-unlock <assertion-id>` subcommand removes an active lock from the configured backend regardless of holder, with a confirmation prompt and clear audit output.

## Success Criteria

- ✅ `spekk force-unlock <assertion-id>` subcommand registered in the CLI
- ✅ Before removing the lock, fetches the current `LockInfo` and displays it:
  ```
  Force-unlock will remove this lock:
    assertion: dashboard-loads-fast
    holder:    alice@macbook
    branch:    feature/dashboard
    age:       23m

  Are you sure? Only do this if you're certain the holder is gone. [y/N]
  ```
- ✅ Interactive prompt defaults to "no" — any answer except `y`/`yes` aborts
- ✅ `--yes` / `-y` flag skips the confirmation prompt
- ✅ On confirmation, calls `LockStore.ForceUnlock(assertionID)` and prints:
  ```
  ✓ Force-unlocked dashboard-loads-fast (previously held by alice@macbook)
  ```
- ✅ Logs to stderr a one-line audit record including: timestamp, local user, local hostname, assertion id, previous holder, previous lock id. Format:
  ```
  [force-unlock] 2026-04-08T16:42:03Z user=bob host=bob-laptop assertion=dashboard-loads-fast previous-holder=alice@macbook previous-lock-id=builder-macbook-12345-1706210400
  ```
- ✅ If no lock exists for the assertion: exit code 1, message `no active lock for assertion: {id}`
- ✅ If the backend is unreachable: same clear error as other lock commands
- ✅ Works against all backends (`local`, `file`, `redis`)
- ✅ Integration tests verify: confirmation flow, `--yes` bypass, missing lock, successful removal

## Safety Notes

- Terraform's `force-unlock` is the recovery tool when a builder crashes mid-work. The confirmation prompt exists because force-unlocking an actively-working builder will cause two builders to work on the same assertion concurrently.
- The audit log line is stderr (not stdout) so it does not interfere with JSON/script output but is captured in builder logs.

## Out of Scope

- Remote audit log persistence (just local stderr for MVP)
- Lock history / who force-unlocked when (future feature)
- Bulk `force-unlock --all-stale` (can be added later, but stale locks already auto-recover via TTL so this is low priority)

## Notes

This completes the recovery path. Between TTL expiry (automatic) and `force-unlock` (manual), every stuck-lock scenario has a resolution.
