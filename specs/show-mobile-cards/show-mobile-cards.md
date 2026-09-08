---
id: show-mobile-cards
created: 2026-09-07T18:00:00Z
priority: 2
---

# Mobile card view for the Spec Explorer

The `spekk show` artifact (`internal/show/template.html`) is a fixed two-panel desktop layout: a left tree with search and a "hide completed" toggle, and a right detail panel with rendered markdown and the cross-branch metro map. It has no responsive behavior.

This spec adds a mobile layout that activates at narrow viewports. Instead of the two panels, specs render as full-screen cards in a vertical scroll-snapping deck. Swiping up on a card raises a sheet listing that spec's assertions; tapping an assertion opens its rendered markdown. Search and the existing filters (hide-completed toggle, and the branch filter in cross-branch mode) remain functional over the deck.

The mobile view is browse-first: cards and their assertion sheets are the primary surface, and assertion markdown is reachable one tap in. The cross-branch metro map stays desktop-only.

## Scope

- One HTML file serves both layouts; the switch is a CSS media query at `≤768px`, so resize/rotation re-lays-out live with no reload.
- The desktop layout (`>768px`) is unchanged. Existing `internal/show` output and tests must not move.
- Data model is unchanged — the template already receives every field the card view needs (spec/assertion id, title, status, priority, content, branch, cross-branch contributions).

## Assertions

- `viewport-breakpoint-switches-layout` — media query swaps two-panel for card deck; desktop byte-identical
- `spec-cards-fill-viewport` — each spec is a full-viewport scroll-snapping card
- `swipe-up-reveals-assertions` — swipe up raises the assertion sheet; tap opens assertion markdown
- `search-filters-card-deck` — search box in a sticky bar filters which cards appear
- `filters-apply-to-cards` — hide-completed toggle and branch filter apply to the deck
