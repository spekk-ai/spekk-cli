---
id: search-filters-card-deck
parent: show-mobile-cards
created: 2026-09-07T18:00:00Z
priority: 2
status: done
depends-on: spec-cards-fill-viewport
branch: feature/show-mobile-cards
---

# Search filters which spec cards appear in the deck

Search stays available in the card layout from a bar that does not scroll away, and it narrows the deck using the same matching the desktop tree uses.

## Success Criteria

- A search input is reachable from a bar that stays fixed in view while cards scroll
- Typing filters which specs appear as cards, using the same match logic as the current desktop tree search
- A spec that matches only through one of its assertions still appears as a card, and its matching assertions are surfaced in that card's sheet
- Clearing the search restores the full deck
- When no spec matches, the deck shows an explicit "no matches" state rather than a blank screen

**Tests:** internal/show/show_test.go (`TestTemplateCardDeckSearch`)
