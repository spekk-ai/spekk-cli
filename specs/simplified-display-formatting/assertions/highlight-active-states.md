---
id: highlight-active-states
parent: simplified-display-formatting
created: 2026-01-22T22:37:00Z
priority: 2
status: done
---

# Highlight Active Status States

Failed and in-progress specs should be the most visually prominent to draw attention to active work.

## What Must Be True

Status styling emphasizes items that need attention while keeping completed items subtle.

### In-Progress Status Styling
- `.status-in_progress` uses soft yellow color scheme
- Background: soft yellow (#fef3c7 or similar warm tone)
- Text/border: darker yellow for contrast
- Keeps pulsing animation to indicate active work
- Animation: `pulse-yellow` with warm yellow glow effect

### Failed Status Styling  
- `.status-failed` (if applicable) uses attention-grabbing red
- Should be most prominent of all status states
- Clear visual indicator that requires immediate action

### Completed Status Styling
- `.status-done` remains subtle green
- Less visually prominent than active states
- Indicates completion without demanding attention

### Not Started Status Styling
- `.status-not_started` remains neutral
- Minimal visual weight until work begins

## Success Criteria

- ✅ In-progress items use soft yellow with pulsing animation
- ✅ Failed items (if any) are most visually prominent  
- ✅ Completed items are subtle and don't compete for attention
- ✅ Active states clearly stand out in the interface