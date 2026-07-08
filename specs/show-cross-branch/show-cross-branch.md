---
id: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 2
---

# Cross-Branch / Merge-Preview Mode for `spekk show`

## Overview

`spekk show` generates an HTML spec explorer (`.spekk/index.html`) from the specs
in the current working tree. This feature adds a **cross-branch mode** that shows
the state of specs and assertions across **all branches** — a read-only preview
of what merging each other branch into the current branch would do to the spec
corpus.

The current branch is treated as **"ours"**. Every other branch (local and
remote-tracking) is compared against it so the explorer can surface incoming
additions, clean modifications, conflicts, and deletions before any merge
actually happens.

## Core Purpose: States to Surface

For each spec / assertion, classify and visually distinguish (all keyed to the
current branch = "ours"):

1. **Incoming addition** — exists on another branch but not on ours (a "foreign"
   spec/assertion).
2. **Incoming modification (clean)** — modified on another branch, unchanged on
   ours, so it would merge cleanly. Especially useful for seeing assertion
   **status drift** (e.g. `not_started` → `done` elsewhere).
3. **Conflict** — modified on **both** ours and another branch in a way that does
   not merge cleanly. Must be explicitly called out.
4. **Incoming deletion** — exists on ours but deleted on another branch.

Specs/assertions unchanged across all branches render normally.

## Implementation Approach (Settled Constraints)

- **Shell out to `git`** via `os/exec`, consistent with existing usage in
  `cmd/spekk/main.go` and `internal/agent/loop.go`. Do **not** add `go-git` or
  `libgit2`/cgo dependencies — the single static binary must be preserved.
- The core primitive is **`git merge-tree --write-tree HEAD <branch>`**
  (git ≥ 2.38): an in-memory three-way merge that reports conflicted paths
  **without** touching the working tree or index. This makes conflict detection
  honest rather than approximate.
- Use **three-way comparison against the merge-base** per branch pair
  (base→ours vs base→theirs) to classify add / modify / delete, and merge-tree
  to confirm true conflicts.
- **Read-only guarantee:** the entire flow must never mutate the working tree or
  index. No `checkout`, `merge`, or index-touching commands.
- **git version floor:** detect git version; require ≥ 2.38 for true conflict
  detection, or degrade gracefully to classification-only (add/modify/delete
  without conflict confirmation) on older git, surfacing this clearly.
- **Branch scope:** compare against the union of local branches **and**
  remote-tracking refs (`origin/*`), deduplicated, with a name filter/glob to
  exclude noise.
- **Granularity:** diff at the assertion-**file** level, then roll up to a
  spec-level summary badge. Reuse the **existing parser** (`internal/parser`) to
  turn file content read out of a ref into Spec/Assertion structs — git answers
  the merge questions, the existing parser answers the what's-in-the-file
  questions. No duplicate parsing logic.
- **Collapsed/union view:** N other branches → N merge-tree calls (linear). Each
  spec/assertion carries a list of `(branch, state)` contributions collapsed into
  one view.

## Flag

- Activated by a distinct flag on `show`: **`--cross-branch`**.
- Do **not** reuse `--all-branches` — that flag already exists on `spekk next`
  with an unrelated meaning (ignore the `branch:` frontmatter constraint within
  the current working tree; it never reads git history).
- Accepts an optional branch name filter/glob to exclude noisy/stale branches.

## Out of Scope for v1 (Future Extension)

- **A-vs-B cross-branch conflict detection** — two *other* branches diverging on
  the same spec, independent of ours. This is valuable from an integration-branch
  vantage and is arguably the observer agent's job (its Type 4 spec-conflict
  drift). The v1 data model (per-spec/assertion list of branch contributions)
  **must not preclude** adding this later, but no v1 assertions target it.

## Assertions

See `assertions/` for what must be true. Ordered foundational git/parsing
plumbing → classification → HTML rendering → watch-mode integration.
