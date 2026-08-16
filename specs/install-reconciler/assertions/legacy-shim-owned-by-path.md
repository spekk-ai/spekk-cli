---
id: legacy-shim-owned-by-path
parent: install-reconciler
created: 2026-08-16T00:00:00Z
priority: 1
status: done
depends-on: hash-guard-backup
---

# A Legacy Agent-Shim Path Is Spekk's by Its Path Too

## Description

`migrateLegacy` removes the old coach and builder agent shims — `spekk-coach.<ext>` and `spekk-builder.<ext>` in the host's agent directory — for a user coming from a version before those roles became skills. It reads the file body to confirm the file is spekk's before it removes it, and that has the same weakness as the sniff in `reconcile`: reword the shim and the old file survives, so the host keeps a stale coach agent beside the new coach skill.

The path carries the same statement of ownership. `spekk` wrote that spekk-namespaced name into that directory; nothing else puts a file there.

## Success Criteria

- `migrateLegacy` removes a file at a legacy agent-shim path whatever its body says. It writes `<path>.bak` first and records one warning that names the file and the backup.
- The two guards that are not about content stay: `migrateLegacy` skips a legacy path that is also a desired path (a host that keeps agents and skills in one directory updates that file in place instead), and it skips a stamped file (`reconcile` owns and prunes that one).
- A legacy path with no file there is still not an error.
- A test proves the change: an unstamped file at `<agent dir>/spekk-coach.md` whose body contains none of `spekk prompt `, `You are the spekk`, or `Spekk Dev Loop` is backed up and removed by `Install`.

**Tests:** `internal/install/install_test.go` — `TestInstall_MigratesUnstampedLegacyShim`, `TestInstall_PrunesStampedLegacyShim`, `TestInstall_MigratesCodexSharedPathShim`.
