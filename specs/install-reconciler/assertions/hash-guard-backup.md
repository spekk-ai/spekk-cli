---
id: hash-guard-backup
parent: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 1
status: done
depends-on: reconcile-writes-and-prunes
---

# A Managed Destination Path Is Spekk's, and a Backup Keeps What Was There

## Description

The destination path is the statement of ownership. `spekk install` writes `~/.claude/skills/spekk-dev-loop/SKILL.md` because that is where the skill goes. The reconciler must not re-derive that fact from the file body, because a second source of truth can disagree with the first: change a heading or a sentence in the file, and every unstamped copy in the field stops matching and stays stale with no signal.

So the reconciler decides ownership of a desired path from the path alone. It always brings the file at that path to the current content. The stamp keeps one job only: it tells the reconciler whether it must make a backup first.

## Success Criteria

- No function in `internal/install` decides ownership of a desired or a legacy path by reading file content. `looksLikeSpekkFile` does not exist. (The stamp is still read: it tells a scan which files spekk wrote, and it tells the reconciler whether a backup is due.)
- A backup that spekk wrote is never treated as a managed file. The backup of a managed file the user edited carries the old stamp, so a scan that counted it would back it up again on every install and report it as stale forever. `scanOwned` skips a name in the `<name>.bak` and `<name>.bak.<n>` forms, and one constant holds that suffix for both `backupFile` and the scan.
- For a file that already exists at a desired path, `reconcile` acts on exactly three cases:
  - The on-disk bytes equal the stamped desired content: no write, no backup, no warning (the reconcile stays idempotent).
  - The file is stamped and its body hash agrees with the stamp (it is pristine, from another spekk version): write the new stamped content, with no backup and no warning.
  - Any other case — a stamped file the user edited, or a file with no stamp, whatever its content: write `<path>.bak` first, then write the new stamped content, and record one warning.
- The warning names the file and the backup and says the file was updated. There is one warning wording for this case, not two. It names the backup `backupFile` actually wrote, which is not always `<path>.bak`, so `backupFile` returns that path to its callers.
- `backupFile` still never overwrites an earlier backup: it falls back to `<path>.bak.1`, `<path>.bak.2`, and so on.
- `backupFile` keeps one copy of any one version. When a backup already holds the same bytes, it keeps that file and writes nothing. The prune half sees an edited file again on every install, so a fresh name each run would add a copy of the same content forever.
- A backup keeps the mode of the file it preserves. A private file must not become readable by way of its backup.
- The prune half of `reconcile` is unchanged: an owned file that is not in the desired set and is not pristine is backed up, left in place, and reported in a warning. Repeated installs report it every run and leave exactly one backup.
- `scanOwned` reads only what it can and only what is safe to read. It skips a file it has no permission to open, and it skips anything that is not a regular file. Two of these directories hold the user's own prompts beside spekk's files: one locked-down file must not stop the scan, and a FIFO must not block it forever. A directory it cannot open is skipped for the same reason.
- Tests cover, at a desired path: a byte-identical file (no write), a pristine stamped file from another version (write, no backup), a stamped file edited by hand (backup, then update), and an unstamped file whose body contains none of the old marker strings (backup, then update). Tests also cover both backup name forms, the backup collision fallback, the one-copy rule, the preserved mode, the warning naming the real backup, and a scan that survives an unreadable file.

**Note:** the user-visible cost is that a hand-edited managed file is replaced on the next install. That is deliberate. The edit survives in the `.bak`, and an install that reports success must leave the installed thing current.

**Tests:** `internal/install/reconcile_test.go` — `TestReconcile_UpdatesFileAtDesiredPath` (edited managed file, unstamped file, pristine file from another version), `TestReconcile_WritesPrunesIdempotent` (byte-identical file), `TestReconcile_IgnoresItsOwnBackups`, `TestIsBackupName_CoversBothForms`, `TestBackupFile_NeverOverwritesAnEarlierBackup`, `TestBackupFile_KeepsOneCopyOfTheSameContent`, `TestBackupFile_KeepsTheMode`, `TestReconcile_PruneHalfDoesNotGrowBackups`, `TestReconcile_WarningNamesTheBackupItWrote`, `TestScanOwned_SkipsAFileItCannotRead`; `internal/install/install_test.go` — `TestInstall_MigratesOldDevLoopSkill`.
