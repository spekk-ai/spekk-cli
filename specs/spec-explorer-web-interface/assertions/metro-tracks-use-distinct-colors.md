---
id: metro-tracks-use-distinct-colors
parent: spec-explorer-web-interface
created: 2026-02-25T21:00:00Z
priority: 2
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Tracks Use Distinct Colors and Horizontal Lanes

## What Must Be True

Dependency tracks (lines connecting stations) use distinct colors AND are laid out in separate horizontal lanes to eliminate overlapping paths. Each dependency chain gets its own color and horizontal lane, similar to how subway lines have both distinct colors and separate tracks.

## Success Criteria

- ✅ Each dependency track/line gets a unique color (not all gray/same color)
- ✅ Colors are visually distinct and accessible (sufficient contrast)
- ✅ Color palette uses ordinal progression (e.g., subway line colors)
- ✅ Dependencies leading to different endpoints use different colors
- ✅ Colors are sufficient to trace paths without needing a legend
- ⬜ **Each dependency track gets its own horizontal lane (no overlapping tracks)**
- ⬜ **Tracks are visually separated so individual chains are easy to follow**
- ⬜ **Layout scales gracefully with many parallel dependency chains**

## Current Issue

**Problem (Phase 1 - SOLVED):**
- Multiple dependency chains converge toward "Done"
- When tracks overlapped, they were all the same color
- Lines blended together making dependency chains unclear
- ✅ **FIXED:** Each track now has distinct color

**Problem (Phase 2 - CURRENT):**
- Even with distinct colors, too many parallel tracks overlap vertically
- Creates visual "spaghetti" that's confusing to follow
- Many independent dependency chains shown in same view
- Current layout places assertions at same depth level vertically, causing overlap
- Example: 10+ parallel tracks crossing over each other

**Solution (Track-Based Horizontal Lanes):**
1. **Assign each dependency track to a horizontal lane**
   - Track 0 (blue) → Lane 0 → Y = 80px
   - Track 1 (orange) → Lane 1 → Y = 180px
   - Track 2 (green) → Lane 2 → Y = 280px
   - Etc.

2. **Within each lane, position assertions by dependency depth**
   - Root assertions (no dependencies) → X = 60px
   - Depth 1 assertions → X = 170px
   - Depth 2 assertions → X = 280px
   - Etc.

3. **Result:**
   - Each colored track occupies its own horizontal lane
   - No vertical overlap between different dependency chains
   - Easy to trace individual tracks from start to end
   - Scales to many parallel chains (just adds more lanes)

**Fallback (Hierarchical Graph Layout):**
If track-based lanes don't work well (e.g., shared dependencies between tracks), implement a proper DAG layout algorithm:
- Use Sugiyama layered graph drawing
- Minimize edge crossings
- Proper hierarchical layout that handles complex dependency graphs

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

**Step 1: Assign tracks and colors**
```javascript
function assignTracksAndColors(assertions) {
  // Find terminal assertions (endpoints with no children)
  const terminalAssertions = assertions.filter(assertion =>
    !assertions.some(child => child.dependsOn === assertion.id)
  );

  // Assign track number and color to each terminal
  const assertionToTrack = new Map();
  const assertionToColor = new Map();

  terminalAssertions.forEach((terminal, index) => {
    const trackIndex = index;
    const color = trackPalette[index % trackPalette.length];

    // Trace back and assign all ancestors to this track
    function assignTrack(assertionId) {
      if (assertionToTrack.has(assertionId)) return;

      assertionToTrack.set(assertionId, trackIndex);
      assertionToColor.set(assertionId, color);

      const assertion = assertions.find(a => a.id === assertionId);
      if (assertion?.dependsOn) {
        assignTrack(assertion.dependsOn);
      }
    }

    assignTrack(terminal.id);
  });

  return { assertionToTrack, assertionToColor, terminalAssertions };
}
```

**Step 2: Layout with horizontal lanes**
```javascript
function layoutAssertions(assertions, assertionToTrack) {
  const positions = new Map();
  const laneHeight = 100; // Vertical spacing between lanes
  const depthSpacing = 110; // Horizontal spacing between depth levels
  const startX = 60;
  const startY = 80;

  assertions.forEach(assertion => {
    const trackIndex = assertionToTrack.get(assertion.id) || 0;
    const depth = calculateDependencyDepth(assertion, assertions);

    const x = startX + (depth * depthSpacing);
    const y = startY + (trackIndex * laneHeight);

    positions.set(assertion.id, { x, y });
  });

  return positions;
}
```

**Step 3: Render tracks with colors and positions**
```javascript
// Generate SVG with track-based layout
const { assertionToTrack, assertionToColor } = assignTracksAndColors(assertions);
const positions = layoutAssertions(assertions, assertionToTrack);

// Draw dependency lines
assertions.forEach(assertion => {
  if (assertion.dependsOn) {
    const fromPos = positions.get(assertion.dependsOn);
    const toPos = positions.get(assertion.id);
    const color = assertionToColor.get(assertion.id);

    // Draw line from fromPos to toPos with color
  }
});
```

## Layout Example

Given 3 terminal assertions (A, B, C) with dependencies:
- Track 0 (Blue): D → A
- Track 1 (Orange): E → F → B
- Track 2 (Green): G → C

Layout:
```
Lane 0 (Blue):   [D]────[A]
Lane 1 (Orange): [E]────[F]────[B]
Lane 2 (Green):  [G]────[C]
```

No overlapping tracks, easy to follow each chain horizontally.