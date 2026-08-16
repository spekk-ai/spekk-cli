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
- Each reported line says which of the two conditions it is and shows the install command that fixes it, for example `<path> is from an old layout (run: spekk install --target claude-code)` and `<path> is out of date (run: spekk install --target claude-code)`.
- The returned lines are sorted, and a file is reported at most once.
- `spekk update` prints the warning for `--check` as well as for a real update. Neither path changes any managed file.
- Tests cover: a clean install reports nothing; an installed file edited afterwards is reported as out of date; a stamped file the layout no longer writes is still reported as an old layout; and a target that was never installed is not reported.

**Tests:** `internal/install/reconcile_test.go` — `TestCheckStale`.
