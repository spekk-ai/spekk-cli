---
id: scan-dedup-cross-branch
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
status: not_started
depends-on: branch-state-machine
---

# Scans Never Re-Flag Drift Already Covered by a Live Claim on a Visible Branch

## Description

Dedup is computed from the cross-branch union of observations, using the same machinery (`internal/crossbranch`) that powers `spekk show`'s merge-preview mode. If a visible branch — including a parked one — carries a live claim on a finding, the scan does not create a second observation for it.

## Success Criteria

- Before flagging drift, the scan consults the union of observations across: all visible `observer/*` branches (local and remote-tracking, after `git fetch`) plus main.
- An observation is a **live claim** only on the branch named after it: `observer/<slug>` carries `observations/<slug>.md`. Every branch is cut from `origin/main` and therefore inherits a copy of every observation already merged, and an inherited copy at another finding's branch is not a claim on anything.
- A slug present on main is resolved history and covers nothing, even while its branch is still visible. Presence on main is the authoritative end of a claim, because the frontmatter status is allowed to lag a merge.
- Drift is "already covered" when a live claim has the same `type` and the same `slug` as the candidate; covered drift produces no new observation and no new branch. An overlapping `affected` path is not a match: the `affected` list of a finding names the code and the spec it disagrees with, so two unrelated findings in one file overlap on that file.
- `affected` entries are compared after normalization wherever they are compared, which removes a `:line` or `:line:column` suffix, a leading `./`, repeated slashes, a trailing slash, and any `.` or `..` segment. A directory is deliberately not reduced to the files under it — prefix containment would let one directory-level finding hide every file-level finding beneath it. `.spekk/dont-flag.yaml` matching is what needs this: suppressed drift never becomes an observation, so no branch exists to cover it on the next run.
- Parked branches (branch exists, PR closed) participate exactly like pending ones — closing a PR without deleting the branch keeps the finding claimed.
- Deleting an `observer/<slug>` branch removes it from the union: a subsequent scan of still-present drift produces a fresh observation. This is the intended "forgotten" path, not a bug.
- The union is built via `internal/crossbranch` (branch discovery + reading files from refs), not by checking branches out or by shelling to a forge API.
- Dedup never compares an artifact against itself: the reference set is observations on *branches/main*, never a digest or summary file produced by the same run — the self-referential digest comparison that caused the production silent failure is structurally impossible.

**Note:** resolved drift that recurs is new drift. A merged finding stays in the union as history, and history answers one question only — `ResolveSlug` gives the recurrence a dated slug so its branch does not collide with the old one.

**Tests:** internal/observation/union_test.go, internal/observation/covers_test.go, cmd/spekk/observer_test.go (TestScanCheckCoveredAndClear, TestScanCheckSlugCollisionWithMain)
