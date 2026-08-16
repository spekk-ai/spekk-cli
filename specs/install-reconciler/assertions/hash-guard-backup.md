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

- No function in `internal/install` decides ownership by reading file content. `looksLikeSpekkFile` does not exist.
- For a file that already exists at a desired path, `reconcile` acts on exactly three cases:
  - The on-disk bytes equal the stamped desired content: no write, no backup, no warning (the reconcile stays idempotent).
  - The file is stamped and its body hash agrees with the stamp (it is pristine, from another spekk version): write the new stamped content, with no backup and no warning.
  - Any other case — a stamped file the user edited, or a file with no stamp, whatever its content: write `<path>.bak` first, then write the new stamped content, and record one warning.
- The warning names the file and the backup and says the file was updated. There is one warning wording for this case, not two.
- `backupFile` still never overwrites an earlier backup: it falls back to `<path>.bak.1`, `<path>.bak.2`, and so on.
- The prune half of `reconcile` is unchanged: an owned file that is not in the desired set and is not pristine is backed up, left in place, and reported in a warning.
- Tests cover, at a desired path: a byte-identical file (no write), a pristine stamped file from another version (write, no backup), a stamped file edited by hand (backup, then update), and an unstamped file whose body contains none of the old marker strings (backup, then update).

**Note:** the user-visible cost is that a hand-edited managed file is replaced on the next install. That is deliberate. The edit survives in the `.bak`, and an install that reports success must leave the installed thing current.

**Tests:** `internal/install/reconcile_test.go` — `TestReconcile_UpdatesFileAtDesiredPath` (edited managed file, unstamped file, pristine file from another version) and `TestReconcile_WritesPrunesIdempotent` (byte-identical file); `internal/install/install_test.go` — `TestInstall_MigratesOldDevLoopSkill`.
