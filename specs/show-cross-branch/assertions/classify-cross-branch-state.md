---
id: classify-cross-branch-state
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: done
depends-on: branch-discovery
---

# Each Spec/Assertion File Is Classified Into One of the Four Cross-Branch States

## Description

For every spec/assertion file, and for each comparison branch, the file is
classified relative to the current branch using a three-way comparison against
the merge-base plus `merge-tree` for true conflict confirmation. This is the
core classification engine the rest of the feature renders.

## Success Criteria

- For each comparison branch, the merge-base with the current branch is found via
  `git merge-base`, and changes are classified by comparing base→ours vs
  base→theirs at the **assertion-file** level.
- A file is classified as exactly one of:
  - **incoming-addition** — present on the other branch, absent on ours.
  - **incoming-modification** — modified on the other branch, unchanged on ours
    (would merge cleanly).
  - **conflict** — modified on **both** ours and the other branch in a way that
    `git merge-tree --write-tree HEAD <branch>` reports as a conflicted path.
  - **incoming-deletion** — present on ours, deleted on the other branch.
- A file unchanged across a branch pair carries no contribution (renders normally
  for that pair).
- Conflict classification uses `merge-tree`'s reported conflicted paths when
  available (git ≥ 2.38). In degraded mode (per `git-version-detection`), a
  both-sides-modified file is classified as a *potential* conflict and labeled as
  unconfirmed rather than asserted as a true conflict.
- Status drift is captured: a clean incoming-modification that only changes an
  assertion's `status` (e.g. `not_started` → `done`) is detectable from the
  parsed structs so the renderer can highlight it.
- Classification is computed by shelling out to `git` (no go-git / libgit2),
  preserving the single static binary.
