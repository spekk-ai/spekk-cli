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
- One function, `inspect`, is the single rule for every read of a managed path, and `scanOwned`, `reconcile`, `migrateLegacy`, and `CheckStale` all go through it. It answers with one of four states: nothing is there, a regular file spekk read, a symlink another tool made, or a path spekk cannot read.
- A path spekk cannot read is not a failure of the run. `inspect` gives that state, and no error, for a path it has no permission to open — the parent directory can be the closed one — and for anything that is not a regular file, because a read of a FIFO would never return. Two of these directories hold the user's own prompts beside spekk's files, so one locked-down file must not stop the whole run. A directory `scanOwned` cannot open is skipped for the same reason.
- `reconcile` and `migrateLegacy` leave such a path alone and warn. `scanOwned` does not own it. A write there would replace something spekk cannot read, and a write to a FIFO would never return.
- `backupFile` never fails the install over a backup it cannot compare. A candidate name that holds a file it cannot read, a directory, or a symlink is simply taken, and the next name is used. A backup is never written through a symlink another tool made.
- `backupFile` stops when it cannot look at a candidate name at all. A name too long stays too long, so counting up to the next name would never end. That case returns an error rather than a loop.
- A warning about a backup says "kept a copy in", not "wrote". A backup that already holds these bytes is reused, and nothing is written, so "wrote" would not be true on a second install over the same edit.
- A path whose parent is a file, not a directory, is nothing to read rather than a failure. A regular file where one tool's config directory belongs must not stop the run for every other tool.
- Tests cover, at a desired path: a byte-identical file (no write), a pristine stamped file from another version (write, no backup), a stamped file edited by hand (backup, then update), and an unstamped file whose body contains none of the old marker strings (backup, then update). Tests also cover both backup name forms, the backup collision fallback, the one-copy rule, the preserved mode, the warning naming the real backup, and a scan that survives an unreadable file.

**Note:** the user-visible cost is that a hand-edited managed file is replaced on the next install. That is deliberate. The edit survives in the `.bak`, and an install that reports success must leave the installed thing current.

**Tests:** `internal/install/reconcile_test.go` — `TestReconcile_UpdatesFileAtDesiredPath` (edited managed file, unstamped file, pristine file from another version), `TestReconcile_WritesPrunesIdempotent` (byte-identical file), `TestReconcile_IgnoresItsOwnBackups`, `TestIsBackupName_CoversBothForms`, `TestBackupFile_NeverOverwritesAnEarlierBackup`, `TestBackupFile_KeepsOneCopyOfTheSameContent`, `TestBackupFile_KeepsTheMode`, `TestReconcile_PruneHalfDoesNotGrowBackups`, `TestReconcile_WarningNamesTheBackupItWrote`, `TestScanOwned_SkipsAFileItCannotRead`, `TestScanOwned_SkipsAFIFO`, `TestScanOwned_SkipsADirectoryItCannotRead`, `TestBackupFile_SurvivesABackupItCannotCompare`, `TestBackupFile_SurvivesADirectoryInItsWay`, `TestBackupFile_DoesNotWriteThroughASymlink`, `TestReconcile_LeavesAPathItCannotRead`, `TestReconcile_PruneWarningNamesTheBackupItWrote`, `TestBackupFile_StopsOnANameItCannotLookAt`, `TestScanOwned_SkipsAFileWhereADirectoryBelongs`, `TestInspect_TreatsAFileWhereADirectoryBelongsAsAbsent`; `internal/install/install_test.go` — `TestInstall_MigratesOldDevLoopSkill`, `TestInstall_LegacyWarningNamesTheBackupItWrote`, `TestInstall_LeavesALegacyPathItCannotRead`.
