---
id: metro-tracks-use-distinct-colors
parent: spec-explorer-web-interface
created: 2026-02-25T21:00:00Z
priority: 2
status: not_started
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Tracks Use Distinct Colors

## What Must Be True

Dependency tracks (lines connecting stations) use distinct, ordinal colors that make overlapping paths easy to distinguish and follow visually.

## Success Criteria

- ✅ Each branch gets a unique track color (not all gray/same color)
- ✅ Colors are visually distinct and accessible (sufficient contrast)
- ✅ Color palette uses ordinal progression (e.g., subway line colors)
- ✅ Overlapping tracks remain distinguishable due to color differences
- ✅ Legend shows branch name → track color mapping
- ✅ Main branch uses neutral color (gray), feature branches use distinct colors

## Current Issue

**Problem:**
- Tracks are hard to follow when they overlap
- Lines blend together making dependency chains unclear
- Difficult to trace which station belongs to which branch

**Solution:**
- Assign distinct colors per branch
- Use standard metro/subway color palette
- Minimum: distinguish main vs feature branches
- Better: each feature branch gets unique color

## Color Palette

**Suggested colors (accessible, distinct):**
- Main branch: `#94a3b8` (neutral gray)
- Feature branches (cycle through):
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
- Consistent stroke-width (2-3px) for clarity

## Implementation

**Assign colors during rendering:**
```javascript
const branchColors = {
  'main': '#94a3b8',
  // Cycle through palette for feature branches
};

function getBranchColor(branchName) {
  if (!branchColors[branchName]) {
    const palette = ['#3b82f6', '#f97316', '#10b981', '#a855f7', '#ec4899', '#14b8a6'];
    const index = Object.keys(branchColors).length % palette.length;
    branchColors[branchName] = palette[index];
  }
  return branchColors[branchName];
}
```

**Apply to SVG paths:**
```javascript
// When rendering track lines
path.setAttribute('stroke', getBranchColor(assertion.branch || 'main'));
path.setAttribute('stroke-width', '3');
```

**Legend:**
- Show colored squares/lines next to branch names
- Positioned above or beside metro map
- Format: `■ feature/auth` with colored square

## Alternative: Reduce Overlap

If colors alone aren't sufficient, consider layout improvements:
- Increase vertical spacing between branch tracks (100px → 150px)
- Adjust curve radius to reduce crossing
- Stagger x-positions slightly for parallel dependencies

However, distinct colors should be the primary solution as they work regardless of layout density.