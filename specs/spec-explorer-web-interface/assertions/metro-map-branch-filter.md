---
id: metro-map-branch-filter
parent: spec-explorer-web-interface
created: 2026-02-25T18:32:00Z
priority: 2
status: not_started
depends-on: metro-map-layout-algorithm
branch: feature/dependency-visualization
---

# Metro Map Branch Filter

## What Must Be True

The metro map view includes a branch filter dropdown that allows users to show/hide specific branches.

## Success Criteria

- ✅ Dropdown appears above the metro map SVG
- ✅ Options: "All branches" + checkbox list of branch names
- ✅ Default: all branches visible
- ✅ Unchecking a branch hides its track and stations
- ✅ Dependency lines that cross to hidden branches become dashed/ghost lines
- ✅ Main branch cannot be hidden (grayed out or always checked)
- ✅ Filter state updates SVG visibility without regenerating
- ✅ Legend updates to reflect visible branches only

## Example UI

```
┌───────────────────────────────────┐
│ Show branches:                    │
│ ☑ main                            │
│ ☑ feature/chat-system             │
│ ☐ feature/authentication          │
│ ☑ feature/ui-redesign             │
└───────────────────────────────────┘

[Metro map shows main + chat + ui-redesign]
[Authentication track hidden]
```

## Visual Behavior

When branch hidden:
- Track line opacity: 0 or display: none
- Station dots on that branch: hidden
- Dependency lines TO hidden branch: dashed stroke, opacity 0.3
- Labels for hidden branch: hidden

## Implementation Notes

- Use CSS classes for visibility: `.branch-main`, `.branch-chat`, etc.
- JavaScript toggles classes based on checkbox state
- Preserve layout (don't recalculate positions)
- Store filter state in browser localStorage (optional enhancement)
