# Spekk CLI 1.2.1 — Dependency Visualization

## Spec Explorer Dependency Visualization (PR #31)

`spekk show` now renders an interactive **metro map** of dependency trees for each branch.

### Metro Map

When you click an assertion, the detail panel shows a collapsible dependency graph at the top. Each branch gets its own map showing how assertions relate to each other.

**Layout:** Independent trees are stacked vertically using a recursive tree-stacking algorithm. Each tree is a self-contained unit — branching nodes fan their children out vertically and sit at the vertical center. This makes overlap between trees structurally impossible.

**Interaction:**
- Click stations to navigate between assertions (no SVG re-render)
- Drag to pan the map in any direction
- Scroll to navigate vertically (shift+scroll for horizontal)
- Resize the map panel with the drag handle
- Collapse/expand the map section (state persists in localStorage)
- Hover stations for full-title tooltips

### New Assertions (11)

All on `feature/dependency-visualization`:

- `branch-metro-map-in-detail-panel` — Collapsible metro map in detail panel
- `clickable-metro-map-stations` — Click-to-navigate stations
- `metro-map-pan-and-zoom` — Drag panning and scroll navigation
- `metro-map-scrollable-viewport` — Resizable viewport with drag handle
- `metro-map-station-tooltips` — Hover tooltips with full titles
- `metro-stations-have-stable-hover-state` — No jitter on hover
- `metro-tracks-use-distinct-colors` — Tree-stacking layout with uniform gray lines
- `subtle-track-terminus-dots` — Endpoint dots for terminal assertions
- `terminal-nodes-have-no-extra-lines` — Clean terminal node presentation
- `hide-metro-map-for-ungrouped-assertions` — Hide map when no dependencies
- `hide-completed-specs-toggle` — Toggle to show/hide completed specs

### Other Changes

- **Locked-by field** — Parser supports `locked-by` field for parallel builder coordination
- **Branch-scoped dependency trees** — Spec parser outputs dependency tree and branch assignments
- **Dead code cleanup** — Removed unused Sugiyama layout functions, track color palette, and other dead code

### Spec Parser Enhancements

All spec-parser assertions now include `depends-on` and `branch` fields, enabling the dependency visualization to render their relationships.

---

## Quick Test

```bash
# Run the explorer
spekk show

# Click any assertion on a feature branch to see the metro map
# Try the spec-parser branch for a complex dependency tree
```

---

## Breaking Changes

None. All changes are additive.
