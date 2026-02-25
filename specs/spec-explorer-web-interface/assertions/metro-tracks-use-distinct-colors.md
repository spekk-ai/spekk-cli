---
id: metro-tracks-use-distinct-colors
parent: spec-explorer-web-interface
created: 2026-02-25T21:00:00Z
priority: 1
status: failed
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Uses Tree-Stacking Layout with Distinct Track Colors

## What Must Be True

Independent dependency trees are laid out as complete, self-contained units stacked vertically. Each tree occupies its own vertical band — no two trees share Y-space, making visual overlap between trees structurally impossible. Within a tree that has internal branching (one parent with multiple children), branches fan out vertically with enough space allocated so they never collide with adjacent trees.

Dependency tracks use distinct colors from an 8-color palette so individual paths are visually traceable.

## Success Criteria

- Independent trees are identified by their root nodes (assertions with no `dependsOn`)
- Each independent tree is rendered as a complete unit before the next tree begins
- Trees are stacked vertically — tree N's entire vertical extent is above tree N+1's start
- Within a branching tree, children of a shared parent each get their own Y row
- The shared parent node sits at the vertical center of its children's Y range
- No two nodes from different trees ever occupy the same Y position
- Horizontal position reflects dependency depth (left-to-right flow)
- No dead/unused layout code exists in the codebase (e.g., unused Sugiyama functions)
- Each dependency track/line gets a color from the 8-color palette
- Colors are visually distinct and accessible (sufficient contrast)
- Dependencies leading to different terminal assertions use different colors

## Color Palette

Track colors (accessible, distinct):
- Blue: `#3b82f6`
- Orange: `#f97316`
- Green: `#10b981`
- Purple: `#a855f7`
- Pink: `#ec4899`
- Teal: `#14b8a6`
- Yellow: `#eab308`
- Red: `#ef4444`

## Failure Mode (Current Bug)

The current layout traces each terminal assertion back to its root independently, placing all nodes in that chain at the same Y. When multiple terminals share a common ancestor (e.g., `parses-frontmatter` has 6 children that are each terminals or lead to terminals), `positions.set()` overwrites the shared node's position with whichever terminal processes it last. This causes streams to visually overlap because nodes end up at Y positions that don't match their connecting lines.
