---
id: metro-map-pan-and-zoom
parent: spec-explorer-web-interface
created: 2026-02-25T20:31:00Z
priority: 1
status: done
depends-on: metro-map-scrollable-viewport
branch: feature/dependency-visualization
---

# Metro Map Pan and Zoom

## What Must Be True

The metro map column supports click-and-drag panning in both directions. Pan bounds are computed from the actual rendered tree layout height so all nodes are reachable with generous margin.

## Success Criteria

- Click and drag anywhere on metro map column to pan
- Panning works in both horizontal and vertical directions
- Cursor changes to grab/grabbing during drag
- Clicking stations still works (doesn't trigger pan)
- **Pan bounds computed from actual tree layout dimensions after all nodes are positioned**
- **Bottom nodes fully visible when panned to lowest position**
- **All edges (top, left, right, bottom) have ≥60px breathing room beyond outermost nodes**
- Pan state preserved when clicking stations (no reset)
- Optional: Mouse wheel for vertical scrolling
- Optional: Pinch-to-zoom on trackpad

## Bound Calculation

Pan limits must be derived from the actual SVG content dimensions, not hardcoded:

```javascript
function calculatePanBounds(mapContainer, svg) {
  const containerRect = mapContainer.getBoundingClientRect();
  const svgWidth = svg.getAttribute('width');
  const svgHeight = svg.getAttribute('height');

  const EDGE_MARGIN = 60; // Extra breathing room beyond content

  return {
    minX: Math.min(0, containerRect.width - svgWidth - EDGE_MARGIN),
    maxX: EDGE_MARGIN,
    minY: Math.min(0, containerRect.height - svgHeight - EDGE_MARGIN),
    maxY: EDGE_MARGIN
  };
}
```

**Critical:** The SVG `width` and `height` attributes must reflect the full computed layout:
```javascript
// After computing all node positions
const maxX = Math.max(...positions.map(p => p.x));
const maxY = Math.max(...positions.map(p => p.y));
const EDGE_PADDING = 60;

const svgWidth = maxX + EDGE_PADDING * 2;
const svgHeight = maxY + EDGE_PADDING * 2;
```

This ensures the SVG is sized to contain all nodes with padding, and pan bounds allow reaching every node.

## Current Bug: Bottom Nodes Cut Off

**Root cause:** SVG height doesn't account for the full tree extent plus padding. When many independent trees stack vertically, the total height can be very large (8000px+) but the pan bounds don't allow scrolling far enough to reveal bottom nodes.

**Fix:** Compute `maxY` from actual positioned nodes after the full layout pass, add `EDGE_PADDING` on all sides, and use these real dimensions for both SVG sizing and pan bound calculation.

## Implementation Notes

- Pan handlers live on the metro map column (third column), not the detail panel
- Use `requestAnimationFrame` for smooth updates
- Pan state is column-level — persists across `showDetail()` calls
- Recalculate bounds when metro map SVG changes (branch switch)
- Ensure clicking stations doesn't trigger pan (check for `.metro-station` target)
