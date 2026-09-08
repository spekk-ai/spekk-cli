---
id: swipe-up-reveals-assertions
parent: show-mobile-cards
created: 2026-09-07T18:00:00Z
priority: 1
status: not_started
depends-on: spec-cards-fill-viewport
branch: feature/show-mobile-cards
---

# Swiping up a card reveals its assertions; tapping one opens its content

A card's assertions live in a sheet that rises from the bottom on an upward gesture. From the sheet, an assertion opens its rendered markdown.

## Success Criteria

- Swiping or dragging up on a card raises a sheet listing that spec's assertions
- Each assertion row in the sheet shows its title, status badge, and priority
- The sheet is dismissable — swiping/dragging down or tapping outside it returns to the collapsed card
- Tapping an assertion row opens that assertion's rendered markdown content
- A back control from the assertion content returns to the assertion sheet
- The interaction responds to both touch gestures and mouse drag/click, so it is exercisable in a desktop browser sized to the mobile breakpoint
- Assertion markdown is rendered with the same renderer the desktop detail panel uses

**Note:** The mobile view is browse-first. The cross-branch metro map is not shown on mobile; only spec cards, assertion sheets, and assertion markdown are reachable.
