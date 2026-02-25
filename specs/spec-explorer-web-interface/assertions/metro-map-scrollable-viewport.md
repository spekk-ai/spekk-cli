---
id: metro-map-scrollable-viewport
parent: spec-explorer-web-interface
created: 2026-02-25T20:00:00Z
priority: 1
status: in_progress
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Column Viewport

**Tests:** src/__tests__/metro-map-scrollable-viewport.test.js

## What Must Be True

The metro map occupies a dedicated right column that takes the full viewport height. Navigation is handled by pan-and-zoom (no scrollbars).

## Success Criteria

- Metro map column takes full viewport height (100vh)
- `overflow: hidden` — no scrollbars visible
- Pan-and-zoom handles all navigation (see metro-map-pan-and-zoom)
- Background: `#f8fafc` with left border separator
- Column width: ~400px fixed
- "Branch Dependencies" header at top showing branch name
- Column collapses or shows notice when metro map is not applicable

## Visual Structure

```
╔══════════════════════════════╗
║ Branch Dependencies          ║
║ feature/dependency-visual... ║
╠══════════════════════════════╣
║                              ║
║  [pan-and-zoom SVG canvas]   ║
║                              ║
║  ○───○───◉───○───●           ║
║       └──○───○───●           ║
║  ○───○───●                   ║
║                              ║
╚══════════════════════════════╝
```

## Implementation Notes

- CSS for metro map column:
  ```css
  .metro-map-panel {
    width: 400px;
    height: 100vh;
    overflow: hidden;
    background: #f8fafc;
    border-left: 1px solid #e2e8f0;
    position: relative;
    cursor: grab;
  }
  ```
- SVG has dynamic width/height based on tree layout dimensions
- Pan-and-zoom handlers attached to `.metro-map-panel`
- When branch changes, replace SVG content and reset pan position
- Column visibility controlled by `shouldShowMetroMap()` logic
