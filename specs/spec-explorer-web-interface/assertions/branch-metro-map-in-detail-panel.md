---
id: branch-metro-map-in-detail-panel
parent: spec-explorer-web-interface
created: 2026-02-25T19:00:00Z
priority: 1
status: not_started
branch: feature/dependency-visualization
---

# Branch Metro Map in Detail Panel

## What Must Be True

When an assertion is clicked in the tree view, its detail panel includes a metro/subway map visualization showing all assertions in the same branch with dependency relationships.

## Success Criteria

- ✅ Detail panel shows metro map section above assertion content
- ✅ Metro map displays only assertions from the clicked assertion's branch
- ✅ Assertions appear as "stations" (colored dots) on a horizontal track
- ✅ Dependency lines connect assertions (parent → child)
- ✅ Clicked assertion is highlighted (thicker stroke, glow, or different color)
- ✅ Assertions positioned left-to-right by dependency depth
- ✅ Labels show assertion titles below each station
- ✅ Pure SVG with minimal JavaScript (no D3 required)
- ✅ If assertion has no branch field (defaults to main), show all main assertions
- ✅ Compact layout fits in detail panel without excessive scrolling

## Visual Structure

When user clicks `chat-message-input`:

```
╔══════════════════════════════════════════════╗
║ Chat Message Input                           ║
║ Status: not_started  Priority: 2             ║
║ Branch: feature/chat-system                  ║
╠══════════════════════════════════════════════╣
║ Branch Dependencies:                         ║
║                                              ║
║  ○─────────○─────────◉─────────○            ║
║  websocket  session  YOU ARE   presence     ║
║  connect    model    HERE      tracking     ║
║                                              ║
╠══════════════════════════════════════════════╣
║ ## What Must Be True                         ║
║ [assertion content...]                       ║
╚══════════════════════════════════════════════╝
```

## Layout Rules

**Horizontal (x) positioning:**
- Assertions with no `depends-on`: x = leftmost
- Assertions with dependencies: x = parent.x + spacing
- Spacing: 100-120px between stations

**Visual styling:**
- Station dots: 8px radius
- Current assertion: 10px radius + glow effect
- Track lines: 4px stroke
- Use color from existing status badge system
- Curved connectors when dependencies branch/merge

## Implementation Notes

- Adapt metro map patterns from `~/personal/chess-opening-study/mockups/tree-concept-a-subway.html`
- Generate SVG during HTML generation in `src/show/cli.js`
- Filter assertions: `assertions.filter(a => a.branch === clickedAssertion.branch)`
- Calculate positions using topological sort of dependencies
- Embed SVG in detail panel, between metadata and content sections
- Keep compact: aim for ~200-300px height
