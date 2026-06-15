---
id: cross-branch-data-model
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: not_started
branch: feat/show-cross-branch
depends-on: classify-cross-branch-state
---

# Show Data Carries Per-Item Branch Contributions and a Spec-Level Rollup

## Description

The data piped into the template (`showData` / `showSpec` / `showAssertion` in
`internal/show/show.go`) gains fields representing cross-branch state. Each
spec/assertion carries a **list of `(branch, state)` contributions** collapsed
into one view, and each spec carries a rolled-up summary derived from its
assertions' contributions.

## Success Criteria

- `showAssertion` (and `showSpec` for foreign/deleted specs) gains a field holding
  a list of contributions, each carrying the contributing branch name and the
  classified state (incoming-addition / incoming-modification / conflict /
  incoming-deletion, plus the unconfirmed-conflict variant from degraded mode).
- `showSpec` gains a rolled-up summary field derived from its assertions'
  contributions, suitable for a single spec-level badge (e.g. "has a conflict"
  dominates, then incoming-modification, then addition/deletion).
- The data model represents **N branches collapsed into one view** per item — it
  is a list, not a single state — so a spec/assertion modified cleanly on one
  branch and conflicting on another shows both contributions.
- The shape does **not preclude** the future A-vs-B extension: contributions are
  keyed by branch and not hard-coded to "vs ours", so additional pairwise
  contributions could be added later without a model rewrite.
- When `--cross-branch` is off, these fields are empty/omitted and existing
  consumers of `showData` are unaffected (current `show` output is byte-for-byte
  equivalent to today for the non-cross-branch path, aside from new
  omitempty fields).
