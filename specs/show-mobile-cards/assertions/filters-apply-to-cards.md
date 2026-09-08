---
id: filters-apply-to-cards
parent: show-mobile-cards
created: 2026-09-07T18:00:00Z
priority: 2
status: in_progress
locked-by: builder-MacBook-Pro-local-23135-1788833058
depends-on: spec-cards-fill-viewport
branch: feature/show-mobile-cards
---

# The completed toggle and branch filter apply to the card deck

The existing filter controls keep working in the mobile layout, and they compose with search.

## Success Criteria

- The "hide completed specs" toggle hides completed spec cards from the deck and restores them when re-enabled
- In cross-branch mode, deselecting a branch excludes that branch's contributions from the deck exactly as it does in the desktop view
- Filters and search compose — both are applied together, and a card appears only when it satisfies every active filter and the current search
- When active filters plus search leave no cards, the deck shows the same "no matches" state as an empty search
