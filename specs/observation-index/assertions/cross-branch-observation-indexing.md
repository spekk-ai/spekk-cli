---
id: cross-branch-observation-indexing
parent: observation-index
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: observation-tables-schema
---

# Indexing Reads Observations From All Fetched `observer/*` Branches Plus Main

## Description

The indexer's observation pass covers the full branch union, not just the
working tree: every visible `observer/*` branch (local and remote-tracking)
plus main contributes rows, read directly from git refs without checkouts.

## Success Criteria

- The observation indexing pass enumerates refs via the same branch-discovery
  rules as `internal/crossbranch` (local branches + remote-tracking refs) and
  includes: every ref matching `observer/*`, plus main.
- Observation files are read from refs (git object reads, as
  `internal/crossbranch` does for specs), never by checking out branches —
  indexing leaves the working tree untouched.
- The same slug appearing on multiple refs (e.g. merged to main and its
  branch not yet deleted) yields one row per ref, distinguished by the ref
  column — the index records what git shows, and consumers decide precedence.
- Indexing performs no remote operations itself. It reads whatever refs are
  already visible; keeping remote-tracking refs current is the caller's job
  (`git fetch` — the announce subcommand does this, per
  `specs/observer-announce/`).
- The staleness/auto-rebuild mechanism from `specs/sqlite-index/` accounts
  for observation sources: a rebuild after new observer branches are fetched
  picks up their observations.

**Note:** ref enumeration must not double-count a branch visible both locally
and as a remote-tracking ref for the same tip; `internal/crossbranch` already
handles this dedup for `spekk show` — reuse it rather than re-deriving the
rules.

**Tests:** internal/index/observation_test.go
(TestObservationSameSlugMultipleRefs,
TestEnsureFreshRebuildsOnNewObserverBranch),
internal/observation/union_test.go (TestLoadUnionSeesRemoteTrackingRefs)
