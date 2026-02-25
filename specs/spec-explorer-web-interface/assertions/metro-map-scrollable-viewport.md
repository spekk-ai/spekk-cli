---
id: metro-map-scrollable-viewport
parent: spec-explorer-web-interface
created: 2026-02-25T20:00:00Z
priority: 1
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Scrollable Viewport

**Tests:** src/__tests__/metro-map-scrollable-viewport.test.js

## What Must Be True

The metro map has a limited height with horizontal scrolling/panning for branches with many assertions.

## Success Criteria

- ✅ Metro map container has max-height (e.g., 300px)
- ✅ Horizontal overflow scrolls/pans when content exceeds viewport width
- ✅ Vertical overflow hidden (no vertical scroll)
- ✅ Visual indicators show when more content exists off-screen (fade edges or scroll hints)
- ✅ Smooth scroll behavior
- ✅ Compact branches fit without scrolling
- ✅ Wide branches (many assertions) scroll horizontally

## Visual Structure

```
╔══════════════════════════════════╗
║ Branch Dependencies:        [→]  ║ ← Scroll hint
║ ┌────────────────────────────┐   ║
║ │ ○───○───○───○───○───○───○ │   ║ ← Scrollable
║ └────────────────────────────┘   ║
╠══════════════════════════════════╣
```

## Implementation Notes

CSS:
```css
.metro-map-container {
  max-height: 300px;
  overflow-x: auto;
  overflow-y: hidden;
  position: relative;
}

.metro-map-container::after {
  /* Fade edge to indicate more content */
  content: '';
  position: absolute;
  right: 0;
  top: 0;
  height: 100%;
  width: 40px;
  background: linear-gradient(to right, transparent, white);
  pointer-events: none;
}
```

- SVG should have dynamic width based on assertion count
- Container scrolls horizontally when SVG width > viewport width
- Consider adding pan/drag interaction (optional enhancement)
