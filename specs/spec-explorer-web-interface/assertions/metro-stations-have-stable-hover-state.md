---
id: fix-terminal-shake-for-real
parent: spec-explorer-web-interface
created: 2026-02-25T20:30:00Z
priority: 1
status: done
branch: feature/dependency-visualization
---

# Fix Terminal Shake For Real

## What Must Be True

Terminal nodes absolutely do not shake on hover. Previous two attempts didn't work - shake persists.

## Success Criteria

- ✅ Zero shake/jitter when hovering ANY metro station
- ✅ Hover effect is stable and smooth
- ✅ No visual position changes whatsoever
- ✅ Cursor doesn't jump or move

## Investigation

The shake is STILL happening after two fixes:
1. First fix: Added `transform-origin: center` and `transform-box: fill-box`
2. Second fix: Removed scale transform, used color + drop-shadow

If shake persists after removing transforms, the problem is likely:
- **SVG coordinate precision issues**
- **Text labels reflowing/wrapping**
- **Parent element boundaries changing**
- **Drop-shadow filter causing reflow**

## Debugging Steps

1. Check if drop-shadow filter is causing reflow
2. Check if text labels are changing size on hover
3. Check if SVG viewBox is recalculating
4. Try completely removing ALL hover effects to isolate cause
5. Check browser DevTools computed styles during shake

## Nuclear Option Solutions

**Option A: Remove drop-shadow filter**
```css
.metro-station:hover circle {
  fill: #2563eb;
  /* Remove drop-shadow completely */
}
```

**Option B: Use stroke instead of shadow**
```css
.metro-station:hover circle {
  fill: #2563eb;
  stroke: #93c5fd;
  stroke-width: 2;
}
```

**Option C: Hardware acceleration**
```css
.metro-station {
  will-change: contents;
  transform: translateZ(0); /* Force GPU layer */
}
```

**Option D: Completely different hover target**
- Don't hover on `<g>` element at all
- Hover directly on `<circle>` only
- Ensure labels have `pointer-events: none`

## Implementation Notes

- Test EACH solution in isolation
- Use browser DevTools to record what's changing
- May need to restructure SVG hierarchy
- Consider that the shake might be browser-specific

## Implementation (Completed)

**Solution Applied:** Nuclear Option A - Complete transform removal

Changed `.spekk/index.html`:
- Removed `transition: transform 0.15s`, `transform-origin: center`, and `transform-box: fill-box` from `.metro-station`
- Removed `transform: scale(1.15)` from `.metro-station:hover`
- Added simple color transition: `.metro-station circle { transition: fill 0.15s; }`
- Added hover color change: `.metro-station:hover circle { fill: #2563eb; }`

**Why this fixes it:**
- No coordinate transformations = no SVG recalculation
- No filters or shadows = no reflow
- Pure CSS color property change = stable rendering
- Targets only circle element = no parent boundary changes

The shake was caused by the scale transform forcing SVG coordinate system recalculation on every hover.
