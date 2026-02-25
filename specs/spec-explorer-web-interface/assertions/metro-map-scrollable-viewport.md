---
id: metro-map-scrollable-viewport
parent: spec-explorer-web-interface
created: 2026-02-25T20:00:00Z
priority: 1
status: in_progress
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Collapsible & Resizable Viewport

**Tests:** src/__tests__/metro-map-scrollable-viewport.test.js

## What Must Be True

The metro map section at the top of the detail panel has a constrained height with pan-and-zoom for navigation, can be collapsed to a thin header bar, and is resizable via a drag handle on the divider.

## Success Criteria

- Metro map section has default height ~300px when expanded
- `overflow: hidden` — no scrollbars, pan-and-zoom handles navigation
- Background: `#f8fafc` with border separating it from content below
- Full width of detail panel (~800px)
- Collapsible header bar with toggle: "▼ Branch Dependencies — {branch-name}"
- Collapsed state: ~36px header-only bar, "▶ Branch Dependencies"
- Collapse/expand has smooth height transition
- Collapsed state persisted in localStorage
- **Drag handle on the divider between metro map and assertion content**
- **Dragging the handle resizes the metro map section height**
- **Height constrained between 100px min and 600px max**
- **Custom height persisted in localStorage**
- **Cursor changes to `ns-resize` on handle hover and during drag**
- **Text selection disabled during drag (prevent accidental highlighting)**
- When metro map is not applicable, section is hidden entirely or shows notice
- SVG has dynamic width/height based on tree layout dimensions

## Visual Structure

**Expanded with drag handle:**
```
╔═══════════════════════════════════════════════╗
║ ▼ Branch Dependencies — feature/dep-viz  [−] ║
║ ┌───────────────────────────────────────────┐ ║
║ │ ○───○───◉───○───●                         │ ║
║ │      └──○───○───●   [pan-and-zoom canvas] │ ║
║ │ ○───○───●                                 │ ║
║ └───────────────────────────────────────────┘ ║
╠═══════════╤═══════════════════════════════════╣ ← drag handle
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

## Drag Handle

- Thin bar (~6px) on the bottom edge of the metro map section
- Styled subtly: slightly darker border or a small grip icon (three horizontal dots)
- Cursor: `ns-resize` on hover
- During drag: body cursor locked to `ns-resize`, `user-select: none` on body
- On mouseup: save new height to `localStorage.setItem('spekkMetroMapHeight', px)`
- On page load: restore height from localStorage (fallback to 300px default)
- When collapsed, drag handle is hidden

## Implementation Notes

- CSS for metro map section:
  ```css
  .metro-map-section {
    height: 300px; /* default, overridden by localStorage */
    min-height: 100px;
    max-height: 600px;
    overflow: hidden;
    background: #f8fafc;
    position: relative;
    cursor: grab;
    transition: height 0.3s ease;
  }

  .metro-map-section.collapsed {
    height: 36px;
    min-height: 36px;
    cursor: default;
    overflow: hidden;
  }

  .metro-map-resize-handle {
    height: 6px;
    cursor: ns-resize;
    background: #e2e8f0;
    border-bottom: 1px solid #cbd5e1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .metro-map-resize-handle:hover {
    background: #cbd5e1;
  }
  ```
- Drag JS (~30 lines): mousedown on handle, mousemove updates section height, mouseup saves
- Disable `transition` during drag for responsive feel, re-enable on mouseup
- Pan-and-zoom handlers attached to `.metro-map-section`
- When collapsed, pan-and-zoom and resize are disabled
- When branch changes, replace SVG content and reset pan position
- Visibility controlled by `shouldShowMetroMap()` logic
- Toggle click handler on `.metro-map-header` bar
