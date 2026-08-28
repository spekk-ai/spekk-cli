---
id: branch-discovery
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: done
depends-on: cross-branch-flag
---

# Comparison Branches Are the Deduplicated Union of Local and Remote Refs

## Description

Cross-branch mode compares the current branch ("ours") against every other
branch. The set of branches to compare is the union of local branches and
remote-tracking refs, deduplicated, with the current branch excluded and an
optional filter applied.

## Success Criteria

- The comparison set is the **union** of local branches and remote-tracking refs
  (e.g. `origin/*`), discovered by shelling out to `git` (e.g.
  `git for-each-ref` or `git branch`).
- The set is **deduplicated**: a local branch that matches its own upstream
  remote-tracking counterpart is counted once, not twice.
- The **current branch ("ours") is excluded** from the comparison set.
- The optional branch name filter/glob from `cross-branch-flag` is applied to
  narrow the set; with no filter, all discovered non-current branches are
  included.
- Detached HEAD or a repo with no other branches is handled gracefully (empty
  comparison set → everything renders as normal, no crash).
