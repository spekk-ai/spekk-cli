---
id: metro-tracks-use-distinct-colors
parent: spec-explorer-web-interface
created: 2026-02-25T21:00:00Z
priority: 2
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Tracks Use Distinct Colors with Hierarchical Layout

## What Must Be True

Dependency tracks (lines connecting stations) use distinct colors AND are laid out using a hierarchical graph algorithm that minimizes edge crossings and visual complexity. The layout makes dependency chains clear and easy to follow even with many parallel paths.

## Success Criteria

- ✅ Each dependency track/line gets a unique color (not all gray/same color)
- ✅ Colors are visually distinct and accessible (sufficient contrast)
- ✅ Color palette uses ordinal progression (e.g., subway line colors)
- ✅ Dependencies leading to different endpoints use different colors
- ✅ Colors are sufficient to trace paths without needing a legend
- ⬜ **Hierarchical layout with nodes arranged in layers by dependency depth**
- ⬜ **Edge crossings minimized using Sugiyama-style algorithm**
- ⬜ **Nodes within each layer positioned to reduce edge crossing**
- ⬜ **Layout scales gracefully with complex dependency graphs**

## Current Issue

**Problem (Phase 1 - SOLVED):**
- Multiple dependency chains converge toward "Done"
- When tracks overlapped, they were all the same color
- Lines blended together making dependency chains unclear
- ✅ **FIXED:** Each track now has distinct color

**Problem (Phase 2 - TRIED, FAILED):**
- Even with distinct colors, too many parallel tracks overlap
- Track-based horizontal lanes approach didn't solve the problem
- The issue is more complex than simple lane assignment
- Dependencies can be shared between multiple tracks
- Simple lane assignment doesn't account for edge crossing minimization

**Solution (Sugiyama Hierarchical Layout):**

Use Sugiyama-style layered graph drawing algorithm:

1. **Layer Assignment (Rank Assignment)**
   - Assign each node to a layer based on longest path from source
   - Layer 0: nodes with no dependencies
   - Layer i: nodes whose longest dependency path is i

2. **Crossing Minimization**
   - Within each layer, order nodes to minimize edge crossings
   - Use barycenter heuristic or median heuristic
   - Multiple passes (forward and backward) to optimize

3. **Coordinate Assignment**
   - Assign X coordinates based on layer
   - Assign Y coordinates within layer to minimize edge length
   - Apply spacing constraints for readability

4. **Result:**
   - Hierarchical left-to-right layout
   - Minimized edge crossings
   - Clear visual hierarchy
   - Handles complex DAGs with shared dependencies

## Color Palette

**Track colors (accessible, distinct):**
- Cycle through for each dependency track:
  - Blue: `#3b82f6`
  - Orange: `#f97316`
  - Green: `#10b981`
  - Purple: `#a855f7`
  - Pink: `#ec4899`
  - Teal: `#14b8a6`
  - Yellow: `#eab308`
  - Red: `#ef4444`

**Accessibility:**
- All colors meet WCAG AA contrast on white background
- Distinguishable for common color vision deficiencies
- Consistent stroke-width (3px) for clarity

## Implementation

**Step 1: Layer Assignment (Longest Path)**
```javascript
function assignLayers(assertions) {
  const layers = new Map(); // assertion.id -> layer number

  function getLongestPath(assertionId, visited = new Set()) {
    if (visited.has(assertionId)) return 0; // Cycle detection

    const layer = layers.get(assertionId);
    if (layer !== undefined) return layer;

    const assertion = assertions.find(a => a.id === assertionId);
    if (!assertion || !assertion.dependsOn) {
      layers.set(assertionId, 0);
      return 0;
    }

    visited.add(assertionId);
    const parentLayer = getLongestPath(assertion.dependsOn, visited);
    const myLayer = parentLayer + 1;
    layers.set(assertionId, myLayer);
    return myLayer;
  }

  assertions.forEach(a => getLongestPath(a.id));
  return layers;
}
```

**Step 2: Minimize Crossings (Barycenter Heuristic)**
```javascript
function minimizeCrossings(assertions, layers) {
  // Group assertions by layer
  const maxLayer = Math.max(...layers.values());
  const layerGroups = Array.from({ length: maxLayer + 1 }, () => []);

  assertions.forEach(assertion => {
    const layer = layers.get(assertion.id);
    layerGroups[layer].push(assertion);
  });

  // Sort nodes within each layer by barycenter of neighbors
  for (let i = 1; i <= maxLayer; i++) {
    layerGroups[i].sort((a, b) => {
      const aParent = assertions.find(p => p.id === a.dependsOn);
      const bParent = assertions.find(p => p.id === b.dependsOn);

      if (!aParent && !bParent) return 0;
      if (!aParent) return -1;
      if (!bParent) return 1;

      // Sort by parent position (barycenter)
      const aParentIndex = layerGroups[i-1].indexOf(aParent);
      const bParentIndex = layerGroups[i-1].indexOf(bParent);
      return aParentIndex - bParentIndex;
    });
  }

  return layerGroups;
}
```

**Step 3: Coordinate Assignment**
```javascript
function assignCoordinates(layerGroups, assertionToColor) {
  const positions = new Map();
  const layerSpacing = 150; // X spacing between layers
  const nodeSpacing = 100; // Y spacing between nodes in same layer
  const startX = 60;
  const startY = 80;

  layerGroups.forEach((layer, layerIndex) => {
    const x = startX + (layerIndex * layerSpacing);

    layer.forEach((assertion, nodeIndex) => {
      const y = startY + (nodeIndex * nodeSpacing);
      positions.set(assertion.id, { x, y });
    });
  });

  return positions;
}
```

**Step 4: Complete Layout Function**
```javascript
function layoutWithSugiyama(assertions, assertionToColor) {
  // 1. Assign layers
  const layers = assignLayers(assertions);

  // 2. Minimize crossings
  const layerGroups = minimizeCrossings(assertions, layers);

  // 3. Assign coordinates
  const positions = assignCoordinates(layerGroups, assertionToColor);

  return positions;
}
```

## Sugiyama Algorithm Benefits

- **Handles Complex DAGs:** Works with shared dependencies between tracks
- **Minimizes Crossings:** Uses heuristics to reduce visual complexity
- **Hierarchical:** Clear left-to-right progression shows dependency flow
- **Scalable:** Adapts to any graph structure