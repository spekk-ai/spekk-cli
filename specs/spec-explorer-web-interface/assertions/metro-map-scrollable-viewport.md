---
id: metro-map-scrollable-viewport
parent: spec-explorer-web-interface
created: 2026-02-25T20:00:00Z
priority: 1
status: in_progress
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Collapsible Viewport

**Tests:** src/__tests__/metro-map-scrollable-viewport.test.js

## What Must Be True

The metro map section at the top of the detail panel has a constrained height with pan-and-zoom for navigation, and can be collapsed to a thin header bar.

## Success Criteria

- Metro map section has max-height (~300px) when expanded
- `overflow: hidden` — no scrollbars, pan-and-zoom handles navigation
- Background: `#f8fafc` with border separating it from content below
- Full width of detail panel (~800px)
- Collapsible header bar with toggle: "▼ Branch Dependencies — {branch-name}"
- Collapsed state: ~36px header-only bar, "▶ Branch Dependencies"
- Collapse/expand has smooth height transition
- Collapsed state persisted in localStorage
- When metro map is not applicable, section is hidden entirely or shows notice
- SVG has dynamic width/height based on tree layout dimensions

## Visual Structure

**Expanded:**
```
╔═══════════════════════════════════════════════╗
║ ▼ Branch Dependencies — feature/dep-viz  [−] ║
║ ┌───────────────────────────────────────────┐ ║
║ │ ○───○───◉───○───●                         │ ║
║ │      └──○───○───●   [pan-and-zoom canvas] │ ║
║ │ ○───○───●                                 │ ║
║ └───────────────────────────────────────────┘ ║
╠═══════════════════════════════════════════════╣
║ [assertion content below]                     ║
╚═══════════════════════════════════════════════╝
```

**Collapsed:**
```
╔═══════════════════════════════════════════════╗
║ ▶ Branch Dependencies — feature/dep-viz  [+] ║
╠═══════════════════════════════════════════════╣
║ [assertion content below]                     ║
╚═══════════════════════════════════════════════╝
```

## Implementation Notes

- CSS for metro map section:
  ```css
  .metro-map-section {
    max-height: 300px;
    overflow: hidden;
    background: #f8fafc;
    border-bottom: 1px solid #e2e8f0;
    position: relative;
    cursor: grab;
    transition: max-height 0.3s ease;
  }

  .metro-map-section.collapsed {
    max-height: 36px;
    cursor: default;
    overflow: hidden;
  }
  ```
- Pan-and-zoom handlers attached to `.metro-map-section`
- When collapsed, pan-and-zoom handlers are disabled
- When branch changes, replace SVG content and reset pan position
- Visibility controlled by `shouldShowMetroMap()` logic
- Toggle click handler on `.metro-map-header` bar
