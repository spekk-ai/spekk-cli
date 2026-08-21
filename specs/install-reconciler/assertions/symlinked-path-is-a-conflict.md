---
id: symlinked-path-is-a-conflict
parent: install-reconciler
created: 2026-08-16T00:00:00Z
priority: 1
status: done
depends-on: hash-guard-backup
---

# A Symlink at a Managed Path Is a Conflict spekk Reports, Not One It Settles

## Description

Ownership by path says spekk writes the file at a managed path. A symlink breaks that. The path is a pointer to a file somewhere else, and that far end is not a path spekk owns. Writing through the link puts spekk's content into another tool's source of truth, usually a dotfiles repository.

Replacing the link starts a loop. The next sync makes the link again and hides spekk's file, and neither side reports it.

Two tools own the path, and only the user can say which one should. So spekk changes nothing and reports the conflict. `spekk install` reports it for every managed path. `spekk update` reports it for a path the current layout writes.

## Success Criteria

- `reconcile` tests a desired path with `os.Lstat` before it reads or writes. When the path is a symlink it writes nothing, makes no `.bak`, and records one warning that names the path and the link target.
- `migrateLegacy` applies the same test before it removes a legacy shim, so an install never quietly deletes a link that another tool made.
- `scanOwned` skips a symlink, so the prune half never removes one either. It reads a regular file only, which covers a link whose far end holds a stamped spekk file.
- `CheckStale` reports a symlinked desired path with its own reason, `StaleSymlink`, and carries the link target in `LinkTarget`. `Remedy()` for that reason asks the user to choose an owner, and never shows an install command.
- `CheckStale` covers a desired path only. A symlink at a legacy path is reported by `spekk install`, on every run, and not by `spekk update`. Legacy paths are a shrinking migration set, so they get the install-time signal alone.
- The test is on the file at the managed path itself. A file inside a symlinked parent directory is an ordinary file and is written as usual. `docs/cli-reference.md` states this limit rather than implying a wider guarantee.
- Tests cover all three guards: a symlink at a desired path (nothing written, link and far end intact, warning names the target), a symlink at a legacy shim path (not removed, warning names the target), and `CheckStale` reporting `StaleSymlink` with no install command.

**Tests:** `internal/install/reconcile_test.go` — `TestReconcile_LeavesASymlinkedPathAlone`, `TestCheckStale_ReportsASymlinkedPath`, `TestScanOwned_SkipsASymlinkToStampedContent`; `internal/install/install_test.go` — `TestInstall_LeavesASymlinkedLegacyShim`; `cmd/spekk/update_report_test.go` — `TestReportStale_SymlinkAsksForAnOwnerAndShowsNoCommand`.
