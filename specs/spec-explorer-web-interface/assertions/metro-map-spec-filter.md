---
id: metro-map-spec-filter
parent: spec-explorer-web-interface
created: 2026-02-25T18:33:00Z
priority: 2
status: not_started
depends-on: metro-map-layout-algorithm
branch: feature/dependency-visualization
---

# Metro Map Spec Filter

## What Must Be True

The metro map view includes a spec filter dropdown that allows users to show only assertions from specific parent specs.

## Success Criteria

- ✅ Dropdown appears above metro map, next to branch filter
- ✅ Options: "All specs" + list of spec names
- ✅ Default: "All specs"
- ✅ Selecting a specific spec filters metro map to show only assertions from that spec
- ✅ Assertions from other specs are hidden (stations + tracks)
- ✅ Dependency lines to hidden assertions become dashed ghost lines
- ✅ Layout remains stable (positions don't shift)
- ✅ Works in combination with branch filter

## Example UI

```
┌─────────────────────────┬──────────────────────┐
│ Filter by spec:         │ Show branches:       │
│ [spec-parser       ▾]   │ ☑ main ☑ feature/x   │
└─────────────────────────┴──────────────────────┘
```

## Use Case

User wants to visualize dependencies for just one feature:
- Select "spec: authentication"
- Metro map shows only auth-related assertions
- Easier to understand complex dependency chains within one feature
- Avoids overwhelming view when many specs exist

## Implementation Notes

- Filter assertions by `parent` field before rendering
- Use CSS classes: `.spec-authentication`, `.spec-parser`, etc.
- JavaScript toggles visibility based on dropdown selection
- Combine with branch filter logic (both filters active simultaneously)
- Keep `depends-on` references visible as ghost lines even if parent is filtered out
