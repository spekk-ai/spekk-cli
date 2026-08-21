---
id: update-reports-outdated-files
parent: install-reconciler
created: 2026-08-16T00:00:00Z
priority: 2
status: done
depends-on: update-warns-on-old-layout
---

# Update Reports an Installed File That Does Not Match This Binary

## Description

A stale installed skill fails quietly. The user keeps running old instructions, and nothing says so: `spekk update --check` reports the binary version only, and `spekk validate` does not look at installed files. The one signal is a line in the output of an install the user has no reason to run.

`spekk update` already knows the embedded content and the install locations, so it can compare them. This turns a silent stale file into something the user sees.

`CheckStale` today reports one condition: an owned file that the current layout no longer writes. It gains a second: a file at a desired path whose bytes do not match what this binary would install.

## Success Criteria

- `CheckStale` takes the skill FS as an explicit parameter — `CheckStale(home, cwd string, skillFS fs.FS)` — instead of reading a package global. `cmd/spekk` passes the embedded FS it already assigns to `install.DefaultSkillFS`.
- For each target and scope, `CheckStale` reports a desired path that **exists on disk** and whose bytes differ from the stamped desired content. A desired path with no file there is not reported, so a target the user never installed stays silent.
- `CheckStale` returns data, not sentences: a `[]StaleFile` of `{Path, Reason, Target, Project, LinkTarget}`, where `Reason` is a closed enum, and `Remedy()` gives the action for that reason. `cmd/spekk` does all the wording. No caller has to parse a string to learn why a file was reported.
- The result is sorted by path, and each path appears at most once. Two scopes can name one path — a user whose working directory is their home directory has the same `.claude` directory in both — and one file with two contradictory fix commands helps nobody. The global scope is checked first, so the global command wins the tie.
- One file is one report even under two spellings. `os.Getwd` returns the path the shell used, so a working directory that reaches home through a symlink names the same `.claude` directory by a second name. The key resolves the symlinks in the parent directory of the path. It never follows a symlink at the path itself, so two managed paths that point at one file stay two reports. `InstalledTargets` keys its scopes the same way.
- `spekk update` prints the warning for `--check` as well as for the already-current case. Neither path changes any managed file.
- A failed check is reported, not swallowed. When `CheckStale` returns an error, `spekk update` prints it as a warning instead of going silent. The scan survives what it cannot read (see `hash-guard-backup`), so this path is for a failure that stops the check as a whole, not for one file another program owns.
- **After a real self-update, the check cannot help**: it runs in the old process, against the old embedded content, so it finds nothing at the exact moment the installed skill goes stale. In that case `spekk update` names the install commands for the targets that have spekk files on disk (`InstalledTargets`) and asks the user to run them.
- The reporting is a function of the install locations and a writer, not of the process environment, so a test can drive it: `reportStale(w, home, cwd)` and `reportReinstall(w, home, cwd)` in `cmd/spekk`. `runUpdate` reads the home and working directories once and passes them in.
- Tests cover: a clean install reports nothing; an installed file edited afterwards is reported as out of date; a stamped file the layout no longer writes is still reported as an old layout; a target that was never installed is not reported; one path shared by two scopes is reported once, with the global command; one path under two spellings is reported once; the symlink report names the link target and gives no install command; the reinstall reminder names the installed targets and stays silent with no install; and a failed check says the check did not run.

**Tests:** `internal/install/reconcile_test.go` — `TestCheckStale`, `TestCheckStale_ReportsEachPathOnce`, `TestCheckStale_ReportsEachPathOnceThroughASymlink`, `TestInstalledTargets_NamesEachScopeOnce`, `TestInstalledTargets_SilentWithNoInstall`; `cmd/spekk/update_report_test.go` — `TestReportStale_SilentOnACleanInstall`, `TestReportStale_SilentWithNoInstall`, `TestReportStale_NamesTheFileAndTheFix`, `TestReportStale_SymlinkAsksForAnOwnerAndShowsNoCommand`, `TestReportReinstall_NamesTheInstalledTargets`, `TestReportReinstall_SilentWithNoInstall`, `TestWarnCheckFailed_SaysTheCheckDidNotRun`.
