---
id: hide-completed-specs-toggle
parent: spec-explorer-web-interface
created: 2026-02-25T19:10:00Z
priority: 2
status: not_started
branch: feature/dependency-visualization
---

# Hide Completed Specs Toggle

## What Must Be True

The spec tree view hides completed specs by default and includes a toggle to show/hide them.

## Success Criteria

- ✅ Specs with `status: done` are hidden by default on page load
- ✅ Toggle checkbox appears in tree panel header: "☐ Show completed specs"
- ✅ Checking toggle reveals all completed specs in tree
- ✅ Unchecking toggle hides completed specs again
- ✅ Toggle state persists in browser localStorage
- ✅ Completed assertions within visible specs remain visible
- ✅ Filter only applies to parent specs (not individual assertions)
- ✅ Tree count updates to reflect filtered view: "X specs, Y assertions (Z completed hidden)"

## Visual Example

**Header with toggle:**
```
┌─────────────────────────────────┐
│ Spec Tree - spekk-cli           │
│ 5 specs, 12 assertions          │
│ (8 completed specs hidden)      │
│                                 │
│ ☐ Show completed specs          │
└─────────────────────────────────┘
```

**Default (completed hidden):**
- spec-parser ✅ → hidden
- builder-agent ✅ → hidden
- authentication ⚠️ → visible (in progress)

**Toggle checked (show all):**
- spec-parser ✅ → visible
- builder-agent ✅ → visible
- authentication ⚠️ → visible

## Behavior Rules

- A spec is considered "completed" when ALL its assertions are done
- Specs with any non-done assertion always visible
- Spec status is computed from assertions (already handled by parser)
- Hide/show applies visual state only (don't re-parse data)

## Implementation Notes

- Use CSS class `.spec-item.completed` with `display: none` by default
- Add checkbox in tree panel header
- JavaScript toggles visibility by adding/removing `.show-completed` class on container
- Store preference: `localStorage.setItem('spekkShowCompleted', 'true/false')`
- Update counter text dynamically based on visible items
- Completed assertions within in-progress specs remain visible (important for context)
