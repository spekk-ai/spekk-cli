---
id: spec-cards-fill-viewport
parent: show-mobile-cards
created: 2026-09-07T18:00:00Z
priority: 1
status: not_started
depends-on: viewport-breakpoint-switches-layout
branch: feature/show-mobile-cards
---

# Each spec is a full-viewport card in a scroll-snapping deck

In the mobile layout every visible spec renders as one card that fills the screen, and the deck snaps so a single card rests in view at a time.

## Success Criteria

- Each non-hidden spec renders as one card that fills the viewport
- The deck uses vertical scroll-snap so exactly one card comes to rest in view at a time
- A collapsed card shows the spec title, its status badge, an `N/M done` assertion count, and its priority
- Card order matches the current desktop tree order (priority first, then the existing sort)
- The card status badge reflects the same computed parent status the desktop tree shows for that spec

**Note:** Card height is `100dvh` (dynamic viewport height), not `100vh`, so mobile browser toolbars do not cause the card to overshoot the visible area.
