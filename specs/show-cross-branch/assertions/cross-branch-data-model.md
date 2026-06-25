---
id: cross-branch-data-model
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: done
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
- A foreign **incoming-addition** spec/assertion — one with no file in the current
  working tree, so it has no locally-parsed entry — is synthesized with its **real
  metadata** (id, title, status, priority, and content) parsed from the branch
  where it exists, reusing `parse-spec-from-ref` rather than being left blank. It
  must **never** carry an empty status or a zero/placeholder priority: a foreign
  item exposes the same fields a local item does, so the renderer can show its
  actual status and priority (e.g. a foreign assertion that is `done` on its
  branch renders as `done`). When the file exists on several branches, any one
  branch's parse is sufficient for the displayed metadata.
- `showSpec` gains a rolled-up summary field derived from its assertions'
  contributions, suitable for a single spec-level badge (e.g. "has a conflict"
  dominates, then incoming-modification, then addition/deletion).
- The data model represents **N branches collapsed into one view** per item — it
  is a list, not a single state — so a spec/assertion modified cleanly on one
  branch and conflicting on another shows both contributions.
- A synthesized **foreign** item (one with no local working-tree file) is flagged
  as such in the data (e.g. a `foreign` boolean), so the client can tell a foreign
  incoming-add apart from a local item and hide it when all of its contributing
  branches are deselected (see `branch-selection-ui`).
- The shape does **not preclude** the future A-vs-B extension: contributions are
  keyed by branch and not hard-coded to "vs ours", so additional pairwise
  contributions could be added later without a model rewrite.
- When `--cross-branch` is off, these fields are empty/omitted and existing
  consumers of `showData` are unaffected (current `show` output is byte-for-byte
  equivalent to today for the non-cross-branch path, aside from new
  omitempty fields).
