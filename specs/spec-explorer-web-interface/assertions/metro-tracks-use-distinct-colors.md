---
id: metro-tracks-use-distinct-colors
parent: spec-explorer-web-interface
created: 2026-02-25T21:00:00Z
priority: 1
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Uses Tree-Stacking Layout with Uniform Gray Lines

## What Must Be True

Independent dependency trees are laid out as complete, self-contained units stacked vertically. Each tree occupies its own vertical band — no two trees share Y-space, making visual overlap between trees structurally impossible. Within a tree that has internal branching (one parent with multiple children), branches fan out vertically with enough space allocated so they never collide with adjacent trees.

All dependency lines (tracks) are uniform gray (`#94a3b8`). There are no per-track colors — visual distinction comes from the spatial layout, not line color.

## Success Criteria

- Independent trees are identified by their root nodes (assertions with no `dependsOn`)
- Each independent tree is rendered as a complete unit before the next tree begins
- Trees are stacked vertically — tree N's entire vertical extent is above tree N+1's start
- Within a branching tree, children of a shared parent each get their own Y row
- The shared parent node sits at the vertical center of its children's Y range
- No two nodes from different trees ever occupy the same Y position
- Horizontal position reflects dependency depth (left-to-right flow)
- No dead/unused layout code exists in the codebase (e.g., unused Sugiyama functions, unused color palette/track-color assignment code)
- All dependency lines use uniform gray color (`#94a3b8`, opacity 0.4, stroke-width 3)
- No per-track color assignment logic exists for dependency lines
