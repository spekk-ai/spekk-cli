---
id: scan-dedup-cross-branch
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: branch-state-machine
---

# Scans Never Re-Flag Drift Already Covered by an Observation on Any Visible Branch

## Description

Dedup is computed from the cross-branch union of observations, using the same
machinery (`internal/crossbranch`) that powers `spekk show`'s merge-preview
mode. If any visible branch — including parked ones — carries an observation
covering a piece of drift, the scan does not create a new observation for it.

## Success Criteria

- Before flagging drift, the scan consults the union of observations across:
  all visible `observer/*` branches (local and remote-tracking, after
  `git fetch`) plus main.
- Drift is considered "already covered" when an existing observation in the
  union has the same `type` and an overlapping `affected` path set for the
  same underlying finding; covered drift produces no new observation and no
  new branch.
- Paths overlap when they name the same file, not when they are the same
  string. `affected` entries are written by an agent, so the same file
  arrives spelled several ways; both sides are compared after normalization,
  which removes a `:line` or `:line:column` suffix, a leading `./`, repeated
  slashes, a trailing slash, and any `.` or `..` segment. A directory is
  deliberately not reduced to the files under it — prefix containment would
  let one directory-level finding hide every file-level finding beneath it.
- The same normalization applies to `.spekk/dont-flag.yaml` matching, which
  needs it more: suppressed drift never becomes an observation, so no branch
  exists to cover it on the next run.
- Parked branches (branch exists, PR closed) participate in the union exactly
  like pending ones — closing a PR without deleting the branch keeps the
  finding suppressed.
- Deleting an `observer/<slug>` branch removes it from the union: a subsequent
  scan of still-present drift produces a fresh observation. This is the
  intended "forgotten" path, not a bug.
- The union is built via `internal/crossbranch` (branch discovery + reading
  files from refs), not by checking branches out or by shelling to a forge
  API.
- Dedup never compares an artifact against itself: the reference set is
  observations on *branches/main*, never a digest or summary file produced by
  the same run — the self-referential digest comparison that caused the
  production silent failure is structurally impossible.

**Note:** observations on main (status `resolved` or `dismissed`) also count
toward the union input, but resolved drift that *recurs* is new drift: if the
code regresses after a merge, the affected paths will show live drift again
and no open observation covers it, so a new observation is correct. The dedup
rule suppresses duplicates of *open/parked* findings, not history.

**Tests:** internal/observation/union_test.go,
cmd/spekk/observer_test.go (TestScanCheckCoveredAndClear,
TestScanCheckSlugCollisionWithMain)
