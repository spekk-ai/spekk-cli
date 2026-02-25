---
id: metro-map-layout-algorithm
parent: spec-explorer-web-interface
created: 2026-02-25T18:31:00Z
priority: 1
status: draft
depends-on: metro-map-dependency-view
branch: feature/dependency-visualization
---

# Metro Map Layout Algorithm

## What Must Be True

The metro map view has a layout algorithm that positions assertions (stations) and draws dependency connections (tracks) in a readable way.

## Success Criteria

- ✅ Assertions are positioned left-to-right based on dependency depth
- ✅ Assertions with no dependencies start at the left (x=0)
- ✅ Assertions that depend on others are positioned to the right of their parent
- ✅ Each branch gets a unique vertical offset (y position)
- ✅ Main branch (`branch: main` or no branch) appears at top
- ✅ Feature branches appear below with increasing vertical offset
- ✅ Tracks (SVG lines) connect parent → child assertions
- ✅ Branch points use curved paths (similar to chess metro map)
- ✅ Stations (circles) are sized consistently with proper spacing
- ✅ Layout handles both linear chains (A→B→C) and parallel work (A→B, A→C)

## Layout Logic

**Horizontal positioning:**
- Level 0: Assertions with no `depends-on` field
- Level 1: Assertions depending on level 0
- Level 2: Assertions depending on level 1
- Spacing: 90-120px between levels

**Vertical positioning:**
- Sort branches alphabetically (main first)
- Assign y-offset: main=80, feature1=180, feature2=280, etc.
- Spacing: 100px between branch tracks

**Track drawing:**
- Straight line for same-branch dependencies
- Curved path when dependency crosses branches

## Example Output

```
Level:     0      1      2      3

main:      A ──── B ──── C
           │             │
branch-1:  │      D      E ──── F
           └─────────────/
```

## Implementation Notes

- Process assertions in dependency order (topological sort)
- Calculate x,y coordinates during layout pass
- Generate SVG elements with calculated positions
- Use Bézier curves for branch connectors (`path` with `C` command)
- Reference chess metro map for curve patterns
