---
id: metro-stations-have-stable-hover-state
parent: spec-explorer-web-interface
created: 2026-02-25T20:30:00Z
priority: 1
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Stations Have Stable Hover State

## What Must Be True

Metro station nodes remain visually stable when hovered - no shake, jitter, or position changes occur.

## Success Criteria

- ✅ Hovering over any metro station produces zero shake or jitter
- ✅ Hover effect is smooth with no visual position changes
- ✅ Cursor remains stable over station dots
- ✅ Only color changes on hover, no transforms

## Implementation

**CSS approach:**
- No transform-based hover effects (removed `transform: scale()`)
- Pure CSS color transition on hover
- No filters or shadows that could cause reflow
- Hover targets only the `<circle>` element

**Why this works:**
- No coordinate transformations = no SVG recalculation
- No layout changes = no reflow
- Simple CSS property change = stable rendering
- Isolated to circle element = no parent boundary changes

**Code:**
```css
.metro-station {
    cursor: pointer;
}

.metro-station circle {
    transition: fill 0.15s;
}

.metro-station:hover circle {
    fill: #2563eb;
}
```
