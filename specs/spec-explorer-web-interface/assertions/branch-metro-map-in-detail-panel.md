---
id: branch-metro-map-in-detail-panel
parent: spec-explorer-web-interface
created: 2026-02-25T19:00:00Z
priority: 1
status: done
branch: feature/dependency-visualization
---

# Persistent Metro Map in Third Column

## What Must Be True

The spec explorer uses a 3-column layout. The metro map lives in a dedicated right column that persists across assertion navigation, showing the dependency graph for the currently selected branch.

## Success Criteria

- ✅ 3-column layout: Tree (left) | Detail (center) | Metro Map (right)
- Container widens beyond 1200px to accommodate three columns
- Metro map column is ~400px wide with its own overflow handling
- Metro map renders once per branch, NOT re-rendered when clicking stations
- Clicking a station updates detail panel and toggles highlight class on map — no SVG re-render
- Pan/zoom state preserved when navigating between assertions in same branch
- When assertion has no applicable metro map (main branch, no deps), right column collapses or shows the no-dependencies notice
- When switching to an assertion on a different branch, metro map re-renders for new branch
- Compact vertical spacing: ~45px between independent trees (down from 70px)
- Labels show assertion titles below each station

## Visual Structure

```
╔═══════════════╦══════════════════════╦══════════════════════╗
║ Tree Panel    ║ Detail Panel         ║ Metro Map Panel      ║
║               ║                      ║                      ║
║ ▼ spec-a      ║ Chat Message Input   ║  ○───○───◉───○       ║
║   assertion-1 ║ Status: not_started  ║  ws   sess YOU pres  ║
║   assertion-2 ║ Priority: 2          ║                      ║
║ ▼ spec-b      ║                      ║  ○───○               ║
║   assertion-3 ║ ## What Must Be True ║  auth  login         ║
║               ║ [content...]         ║                      ║
╚═══════════════╩══════════════════════╩══════════════════════╝
```

## Layout Rules

**Container:**
- Remove `max-width: 1200px` or increase to accommodate 3 columns
- Tree panel: ~300px fixed width
- Detail panel: flex: 1 (fills remaining space)
- Metro map panel: ~400px fixed width, border-left separator

**Metro map column:**
- Full viewport height (minus any header)
- `overflow: hidden` with pan-and-zoom for navigation
- Background: `#f8fafc` (matches current metro map section)
- Pan/zoom state managed as column-level state, not per-assertion

**Compact spacing:**
- Tree spacing (vertical between independent chains): 45px
- Layer spacing (horizontal between dependency levels): 120px
- Station radius: 8px (normal), 10px (selected)

## Implementation Notes

- Generate ONE metro map SVG per branch, embedded in the right column
- On `showDetail()`, update detail panel content AND toggle `.metro-station-current` class on stations
- When assertion's branch changes, regenerate the metro map SVG
- Metro map column gets its own pan-and-zoom handlers (separate from detail panel scroll)
- Right column should have a "Branch Dependencies" header showing the branch name
