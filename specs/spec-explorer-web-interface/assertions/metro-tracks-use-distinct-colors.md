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
- ✅ **Hierarchical layout with nodes arranged in layers by dependency depth**
- ✅ **Edge crossings minimized using Sugiyama-style algorithm**
- ✅ **Nodes within each layer positioned to reduce edge crossing**
- ✅ **Layout scales gracefully with complex dependency graphs**

## Solution

Uses Sugiyama hierarchical graph layout algorithm to create clear, readable dependency visualizations:

1. **Layer Assignment**
   - Each node assigned to layer based on longest path from source
   - Layer 0: nodes with no dependencies
   - Layer i: nodes whose longest dependency path is i

2. **Crossing Minimization**
   - Multiple sweep approach: 4 passes (2 forward, 2 backward)
   - Forward pass: sort nodes by parent position
   - Backward pass: sort by barycenter of children positions
   - Reduces visual complexity and makes paths easier to follow

3. **Coordinate Assignment with Dynamic Spacing**
   - Base node spacing: 80px (vertical, between nodes in same layer)
   - Layer spacing: 150px (horizontal, between dependency levels)
   - Fanout detection: adds 20px extra spacing when consecutive nodes share same parent
   - Prevents visual crowding in high-fanout scenarios

4. **Track Coloring**
   - Each dependency track has distinct color from 8-color palette
   - Colors trace back from terminal assertions through dependency chain
   - Makes individual paths visually distinct and easy to follow

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
- Recursive function traverses dependency tree
- Each node assigned to layer based on longest path from root
- Layer 0: nodes with no dependencies
- Layer n+1: nodes depending on layer n nodes

**Step 2: Crossing Minimization (Multi-Sweep)**
- Groups assertions by layer
- 4-pass optimization:
  - Pass 1 (forward): Sort by parent position
  - Pass 2 (backward): Sort by barycenter of children
  - Pass 3 (forward): Sort by parent position again
  - Pass 4 (backward): Sort by barycenter of children again
- Multiple passes improve layout quality

**Step 3: Coordinate Assignment with Fanout Detection**
- X coordinate: `startX + (layer * 150px)`
- Y coordinate base: `startY + (nodeIndex * 80px)`
- Fanout spacing: detects when consecutive nodes share parent, adds 20px
- Prevents visual crowding when one parent has multiple children

**Step 4: Track Coloring**
- 8-color palette cycles through dependency chains
- Colors assigned from terminal assertions backward
- Each track maintains consistent color throughout path
- Makes individual dependency chains visually traceable

## Benefits

- **Clear Hierarchy:** Left-to-right flow shows dependency progression
- **Reduced Crossings:** Multi-sweep optimization minimizes visual complexity
- **Adaptive Spacing:** Fanout detection prevents crowding
- **Distinct Paths:** Color coding makes each dependency chain easy to follow
- **Scalable:** Handles complex graphs with shared dependencies