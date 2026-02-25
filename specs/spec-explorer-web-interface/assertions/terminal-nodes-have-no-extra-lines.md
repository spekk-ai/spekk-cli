---
id: terminal-nodes-have-no-extra-lines
parent: spec-explorer-web-interface
created: 2026-02-25T20:32:00Z
priority: 2
status: done
depends-on: subtle-track-terminus-dots
branch: feature/dependency-visualization
---

# Terminal Nodes Have No Extra Lines

## What Must Be True

Terminal nodes (assertions with no children) have clean visual presentation with no erroneous lines extending beyond them.

## Success Criteria

- ✅ Terminal nodes have no extra lines extending to their right
- ✅ Track lines only appear between assertions with actual `depends-on` relationships
- ✅ "Done" convergence terminal connections are clean and intentional
- ✅ No stray horizontal lines across the metro map

## Implementation

**Root cause:** A main track line was being drawn horizontally across the entire metro map width, extending beyond all terminal nodes.

**Solution:** Removed the main track line generation. Metro map now only displays:
- Dependency lines between connected assertions (parent → child via `depends-on`)
- Convergence paths from multiple terminal nodes to the "Done" terminus

**Code removed:**
```javascript
// Removed this line that extended across entire map
<line class="metro-track" x1="40" y1="${trackY}" x2="${maxX}" y2="${trackY}"/>
```

The metro map visual is now clean with only meaningful dependency connections visible.
