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
- `CheckStale` returns data, not sentences: a `[]StaleFile` of `{Path, Reason, Target, Project}`, where `Reason` is a closed enum of `StaleOldLayout` and `StaleOutOfDate`, and `InstallCommand()` builds the fix command. `cmd/spekk` does all the wording. No caller has to parse a string to learn why a file was reported.
- The result is sorted by path, and each path appears at most once. Two scopes can name one path — a user whose working directory is their home directory has the same `.claude` directory in both — and one file with two contradictory fix commands helps nobody. The global scope is checked first, so the global command wins the tie.
- `spekk update` prints the warning for `--check` as well as for the already-current case. Neither path changes any managed file.
- A failed check is reported, not swallowed. When `CheckStale` returns an error, `spekk update` prints it as a warning instead of going silent, because one unreadable directory in one tool's config would otherwise disable the whole check.
- **After a real self-update, the check cannot help**: it runs in the old process, against the old embedded content, so it finds nothing at the exact moment the installed skill goes stale. In that case `spekk update` names the install commands for the targets that have spekk files on disk (`InstalledTargets`) and asks the user to run them.
- Tests cover: a clean install reports nothing; an installed file edited afterwards is reported as out of date; a stamped file the layout no longer writes is still reported as an old layout; a target that was never installed is not reported; and one path shared by two scopes is reported once, with the global command.

**Tests:** `internal/install/reconcile_test.go` — `TestCheckStale`, `TestCheckStale_ReportsEachPathOnce`.
