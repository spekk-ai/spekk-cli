---
id: metro-tracks-use-distinct-colors
parent: spec-explorer-web-interface
created: 2026-02-25T21:00:00Z
priority: 2
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Tracks Use Distinct Colors

**Tests:** src/__tests__/metro-track-colors.test.js

## What Must Be True

Dependency tracks (lines connecting stations) use distinct, ordinal colors that make overlapping paths easy to distinguish and follow visually. Each separate dependency chain/line gets its own color, similar to how different subway lines have different colors.

## Success Criteria

- ✅ Each dependency track/line gets a unique color (not all gray/same color)
- ✅ Colors are visually distinct and accessible (sufficient contrast)
- ✅ Color palette uses ordinal progression (e.g., subway line colors)
- ✅ Overlapping tracks remain distinguishable due to color differences
- ✅ Dependencies leading to different endpoints use different colors
- ✅ Colors are sufficient to trace paths without needing a legend

## Current Issue

**Problem:**
- Multiple dependency chains converge toward "Done"
- When tracks overlap, they're all the same color
- Lines blend together making dependency chains unclear
- Difficult to trace which path a station belongs to

**Solution:**
- Assign distinct colors per dependency track/line
- Use standard metro/subway color palette
- Each terminal assertion (endpoint) defines a separate colored track
- All dependencies leading to that terminal use the same color
- Different tracks use different colors

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

**Assign colors per dependency track:**
```javascript
function assignTrackColors(assertions) {
  // Find terminal assertions (endpoints with no children)
  const terminalAssertions = assertions.filter(assertion =>
    !assertions.some(child => child.dependsOn === assertion.id)
  );

  // Assign a color to each terminal
  const assertionToColor = new Map();
  terminalAssertions.forEach((terminal, index) => {
    const color = trackPalette[index % trackPalette.length];
    assertionToColor.set(terminal.id, color);
  });

  // Trace back and color entire dependency chain
  function colorChain(assertionId, color) {
    assertionToColor.set(assertionId, color);
    const assertion = assertions.find(a => a.id === assertionId);
    if (assertion?.dependsOn) {
      colorChain(assertion.dependsOn, color);
    }
  }

  terminalAssertions.forEach((terminal, index) => {
    const color = trackPalette[index % trackPalette.length];
    colorChain(terminal.id, color);
  });

  return assertionToColor;
}
```

**Apply to SVG paths:**
```javascript
// When rendering track lines
const trackColor = assertionToColor.get(assertion.id);
path.setAttribute('stroke', trackColor);
path.setAttribute('stroke-width', '3');
path.setAttribute('opacity', '0.6');
```

## Alternative: Reduce Overlap

If colors alone aren't sufficient, consider layout improvements:
- Increase vertical spacing between branch tracks (100px → 150px)
- Adjust curve radius to reduce crossing
- Stagger x-positions slightly for parallel dependencies

However, distinct colors should be the primary solution as they work regardless of layout density.