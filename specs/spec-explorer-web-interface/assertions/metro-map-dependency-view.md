---
id: metro-map-dependency-view
parent: spec-explorer-web-interface
created: 2026-02-25T18:30:00Z
priority: 1
status: draft
branch: feature/dependency-visualization
---

# Metro Map Dependency View

## What Must Be True

The spec explorer includes a metro/subway map visualization that shows assertion dependencies as a graph, similar to the chess opening tree visualization in `~/personal/chess-opening-study/mockups/tree-concept-a-subway.html`.

## Success Criteria

- ✅ New tab in spec explorer: "Dependencies" (alongside existing "Specs" view)
- ✅ Metro map renders as SVG showing assertion dependency chains
- ✅ Each feature branch gets a unique colored track/line
- ✅ Assertions appear as "stations" (dots) on the tracks
- ✅ Dependency relationships shown as tracks connecting stations
- ✅ Assertions with no dependencies appear on the main line
- ✅ Branch names displayed as labels on colored tracks
- ✅ Clicking a station shows assertion detail in right panel (reuses existing detail panel)
- ✅ Pure SVG + minimal JavaScript (no D3 library required)
- ✅ Legend shows branch colors and meanings

## Visual Structure

```
Main Line (gray): ○────○────○────○
                        │
Feature/chat (blue):    ○────○────○
                                  │
Feature/auth (orange):            ○────○
```

Each "○" is an assertion. Lines connect dependencies. Different branches get different colors and vertical offsets.

## Implementation Notes

- Adapt metro map patterns from chess opening study
- Use existing parser data (`depends-on`, `branch` fields)
- Generate SVG layout with horizontal tracks for each branch
- Curve connectors show when assertion depends on another
- Keep JavaScript minimal (just tab switching + station clicks)
- Reuse existing spec explorer HTML structure and detail panel
