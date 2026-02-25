---
id: fix-terminal-node-hover-shake
parent: spec-explorer-web-interface
created: 2026-02-25T19:20:00Z
priority: 1
status: not_started
branch: feature/dependency-visualization
---

# Fix Terminal Node Hover Shake

## What Must Be True

Terminal nodes in the metro map do not shake or jitter when hovering over them.

## Success Criteria

- ✅ Hovering over terminal nodes (assertions with no children) is stable
- ✅ No visual jitter, shake, or position changes on hover
- ✅ Cursor remains stable over the station dot
- ✅ Hover effects (if any) apply smoothly without layout shifts

## Problem

When hovering over terminal nodes in the metro map, the visual shakes around. This likely indicates:
- Layout recalculation on hover
- Conflicting CSS transforms
- SVG element position changes triggered by hover state

## Implementation Notes

Common causes:
- `transform: scale()` on hover combined with other transforms
- Tooltip positioning causing layout shifts
- Label text wrapping/unwrapping on hover
- Conflicting hover styles on parent/child elements

Solutions to try:
- Use `transform-origin: center` for any scale transforms
- Use `position: absolute` for tooltips (don't affect layout)
- Add `pointer-events: none` to labels if they interfere
- Use `will-change: transform` for smoother animations
- Ensure station circles have fixed dimensions
