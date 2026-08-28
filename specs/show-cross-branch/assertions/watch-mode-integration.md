---
id: watch-mode-integration
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 3
status: done
depends-on: html-state-rendering
---

# Cross-Branch Mode Works Under `--watch` / Serve and Re-Diffs on Change

## Description

When the explorer runs in watch/serve mode (`spekk show -w`), cross-branch mode
keeps the preview current by re-running the classification as branches and specs
change. This is the lowest-priority slice and may ship after the one-shot path.

## Success Criteria

- `spekk show --cross-branch -w` (and `--watch`) runs cross-branch mode under the
  existing watch/serve infrastructure in `internal/show/watch.go`.
- The cross-branch preview is recomputed when relevant git state changes (branch
  refs move, new commits on any branch including the current one), so the rendered
  view reflects the latest branch states without a manual restart. Because the
  classification compares committed refs and never reads the working tree, it is
  cached on a git ref-state fingerprint: an uncommitted working-tree spec edit
  re-renders (reparses and redraws) but correctly reuses the cached classification
  — which a non-committed edit cannot change — while a ref/commit move invalidates
  the cache and reclassifies.
- Re-diffing reuses the same read-only classification path; the
  `read-only-guarantee` assertion still holds in watch mode (no working-tree or
  index mutation on any refresh).
- Recompute cost stays linear in the number of comparison branches (N branches →
  at most N merge-tree calls when git ref state has changed) and is skipped
  entirely when refs are unchanged, so there is no unbounded growth across
  refreshes.
- If watch-mode integration is meaningfully more work than the one-shot path, it
  is acceptable for this assertion to land after the rest of the spec is `done`;
  it does not block the one-shot cross-branch experience.
