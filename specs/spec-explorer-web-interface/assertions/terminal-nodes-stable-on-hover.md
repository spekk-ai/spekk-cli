---
id: fix-terminal-shake-properly
parent: spec-explorer-web-interface
created: 2026-02-25T20:02:00Z
priority: 1
status: done
branch: feature/dependency-visualization
---

# Fix Terminal Shake Properly

## What Must Be True

Terminal nodes do not shake on hover. The previous fix didn't work - terminals still shake erratically.

## Success Criteria

- ✅ Hovering over any metro station (terminal or not) has zero shake/jitter
- ✅ Hover effect is smooth and stable
- ✅ No position changes, only visual changes (scale, color, etc.)
- ✅ Test on multiple browsers if possible

## Problem Analysis

The previous fix added `transform-origin: center` and `transform-box: fill-box`, but shake persists. Possible causes:

1. **Label text causing reflow** - Text below stations might be changing size/wrapping
2. **Pointer events conflict** - Label or track lines intercepting mouse events
3. **SVG coordinate system** - Transform origin not working as expected in SVG context
4. **Multiple transforms** - Scale transform conflicting with translate positioning
5. **Hover on wrong element** - Hover might be on `<g>` but transform on `<circle>`

## Investigation Steps

1. Check if label text has `pointer-events: none`
2. Check if hover transform is on `<g>` (group) vs `<circle>`
3. Try removing scale transform entirely - does shake still happen?
4. Try adding `pointer-events: bounding-box` to `<g>` element
5. Check browser DevTools during hover to see what's changing

## Potential Solutions

**Option A: No scale transform**
```css
.metro-station:hover .station-dot {
  fill: #2563eb; /* Just color change, no scale */
  filter: drop-shadow(0 0 4px rgba(37, 99, 235, 0.5));
}
```

**Option B: Disable pointer events on child elements**
```css
.metro-station text {
  pointer-events: none;
}
.metro-station line {
  pointer-events: none;
}
```

**Option C: CSS-only hover on circle**
```css
.station-dot {
  transition: r 0.15s;
}
.metro-station:hover .station-dot {
  r: 10; /* Change radius via CSS, not transform */
}
```

## Implementation Notes

- Test each solution in isolation
- Use browser DevTools to identify what's changing on hover
- Consider recording video of the shake to analyze frame-by-frame
- May need to restructure SVG element hierarchy
