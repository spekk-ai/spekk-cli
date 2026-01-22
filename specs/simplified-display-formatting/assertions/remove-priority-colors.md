---
id: remove-priority-colors
parent: simplified-display-formatting
created: 2026-01-22T22:35:00Z
priority: 2
status: done
---

# Remove Priority Number Color Coding

Priority numbers should be plain text without color-coded backgrounds or styling.

## What Must Be True

Priority badges display as plain numbers without visual emphasis through colors.

### CSS Priority Styles
- `.priority-1`, `.priority-2`, `.priority-3` classes have no colored backgrounds
- No gradients, borders, or animations on priority badges
- Priority numbers appear as neutral, unstyled text
- Remove `urgent-glow` animation and colored styling

### Plain Number Display
- Priority badges show: `1`, `2`, `3` as plain text
- No red/orange/green color coding
- Consistent neutral styling across all priority levels
- Focus on content, not visual hierarchy through color

## Success Criteria

- ✅ Run `spekk show` and verify priority numbers are not color-coded
- ✅ All priority badges (1, 2, 3) have same neutral appearance  
- ✅ No colored backgrounds, gradients, or animations
- ✅ Priority display is minimal and clean