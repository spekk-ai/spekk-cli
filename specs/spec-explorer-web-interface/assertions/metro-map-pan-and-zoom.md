---
id: metro-map-pan-and-zoom
parent: spec-explorer-web-interface
created: 2026-02-25T20:31:00Z
priority: 1
status: not_started
branch: feature/dependency-visualization
---

# Metro Map Pan and Zoom

## What Must Be True

The metro map supports click-and-drag panning in both horizontal and vertical directions, allowing full navigation of the dependency tree.

## Success Criteria

- ✅ Click and drag anywhere on metro map to pan
- ✅ Panning works in both horizontal and vertical directions
- ✅ Cursor changes to grab/grabbing during drag
- ✅ Smooth panning motion (no lag or jitter)
- ✅ Pan state is constrained (can't pan beyond content bounds)
- ✅ Clicking stations still works (doesn't trigger pan)
- ✅ Optional: Mouse wheel for vertical scrolling
- ✅ Optional: Pinch-to-zoom on trackpad

## Current Problem

Metro map has `overflow-x: auto` for horizontal scroll, but `overflow-y: hidden` for vertical. This means:
- Can scroll horizontally (good)
- Cannot scroll vertically (bad - can't see full tree)
- Scrollbars are clunky UX

## Proposed Solution

Replace scrollbars with click-and-drag panning:

**CSS Changes:**
```css
.metro-map-section {
  max-height: 400px; /* Increase from 300px */
  overflow: hidden; /* Hide both scrollbars */
  cursor: grab;
  position: relative;
}

.metro-map-section.panning {
  cursor: grabbing;
  user-select: none;
}

.metro-map-svg {
  /* SVG will be positioned via transform */
  transition: transform 0.1s ease-out;
}
```

**JavaScript Interaction:**
```javascript
let isPanning = false;
let startX, startY;
let currentX = 0, currentY = 0;

mapContainer.addEventListener('mousedown', (e) => {
  if (e.target.closest('.metro-station')) return; // Don't pan if clicking station
  isPanning = true;
  startX = e.clientX - currentX;
  startY = e.clientY - currentY;
  mapContainer.classList.add('panning');
});

document.addEventListener('mousemove', (e) => {
  if (!isPanning) return;
  currentX = e.clientX - startX;
  currentY = e.clientY - startY;

  // Constrain to bounds
  const bounds = calculateBounds();
  currentX = Math.max(bounds.minX, Math.min(bounds.maxX, currentX));
  currentY = Math.max(bounds.minY, Math.min(bounds.maxY, currentY));

  svg.style.transform = `translate(${currentX}px, ${currentY}px)`;
});

document.addEventListener('mouseup', () => {
  isPanning = false;
  mapContainer.classList.remove('panning');
});
```

## Boundary Calculation

Prevent panning beyond content:
```javascript
function calculateBounds() {
  const containerRect = mapContainer.getBoundingClientRect();
  const svgRect = svg.getBoundingClientRect();

  return {
    minX: Math.min(0, containerRect.width - svgRect.width),
    maxX: 0,
    minY: Math.min(0, containerRect.height - svgRect.height),
    maxY: 0
  };
}
```

## Implementation Notes

- Use `requestAnimationFrame` for smooth updates
- Add momentum/easing for polish (optional)
- Consider adding minimap overview (optional)
- Test with very large dependency trees
- Ensure clicking stations doesn't trigger pan
