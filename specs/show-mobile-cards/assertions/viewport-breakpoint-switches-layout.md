---
id: viewport-breakpoint-switches-layout
parent: show-mobile-cards
created: 2026-09-07T18:00:00Z
priority: 1
status: in_progress
branch: feature/show-mobile-cards
locked-by: builder-MacBook-Pro.local-67604-1788829391
---

# A viewport breakpoint swaps the two-panel layout for the card deck

The Spec Explorer serves both layouts from the one embedded HTML file, choosing between them purely by viewport width.

## Success Criteria

- At viewport width `≤768px` the two-panel layout (tree panel + detail panel) is not displayed and the card deck is displayed instead
- At viewport width `>768px` the existing two-panel layout renders exactly as it does today
- The switch is driven by a CSS media query, so resizing or rotating across the breakpoint re-lays-out live with no reload and no re-run of `spekk show`
- A single generated HTML file serves both layouts — there is no separate mobile artifact or build
- The desktop rendering path is unchanged: existing `internal/show` output and its tests pass without modification

**Note:** The breakpoint is a max-width media query at `768px` (`≤768px` = mobile). Both layouts exist in the same DOM; only their visibility differs by width.
