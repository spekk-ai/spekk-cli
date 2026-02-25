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

Metro station nodes remain visually stable when hovered - no shake, jitter, or position changes occur. Stations are clickable without "running away" from the cursor.

## Success Criteria

- ✅ Hovering over any metro station (including terminals) produces zero shake or jitter
- ✅ Hover effect is smooth with no visual position changes
- ✅ Cursor remains stable over station dots - elements don't "run away"
- ✅ Only color/fill changes on hover, no transform properties
- ✅ Stations with long names remain stable (no text-based reflow)
- ✅ Terminal convergence nodes are stable and clickable

## Current Bug

**Symptoms:**
- Hovering over terminal stations causes rapid shaking/jitter
- Element appears to "run away" from cursor
- Stations become unclickable due to movement

**Root cause:**
- CSS still uses `transform: scale(1.15)` on hover
- Scaling changes SVG bounding box mid-hover
- Creates feedback loop: hover → scale → boundary shifts → no longer hovering → scale back → repeat

## Implementation

**CSS approach (fixed):**
- **Remove all transform-based hover effects** - no `transform: scale()`, no `transform-origin`
- Pure CSS fill/stroke transition on hover
- No filters, shadows, or transforms that could cause reflow
- Target only `circle` fill property, not parent `<g>` element

**Why this works:**
- No coordinate transformations = no SVG recalculation
- No layout/boundary changes = no reflow
- Simple CSS property change = stable rendering
- Isolated to circle fill = no parent boundary changes

**Code:**
```css
.metro-station {
    cursor: pointer;
    /* NO transition, transform-origin, or transform-box properties */
}

.metro-station circle {
    transition: fill 0.15s;
}

.metro-station:hover circle {
    fill: #2563eb; /* Only change fill color */
}

/* Ensure no hover transforms on any metro elements */
.metro-station:hover {
    /* NO transform: scale() or any transform property */
}
```

## Tests

**Manual verification:**
1. Open spec explorer metro map view
2. Hover over various stations, especially terminals at convergence points
3. Verify zero shake/jitter
4. Verify stations remain clickable while hovering
5. Move cursor slowly across station - should remain stable
