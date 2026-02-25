---
id: filter-by-branch
parent: spec-explorer-web-interface
created: 2026-02-25T18:02:00Z
priority: 2
status: draft
---

# Filter by Branch

## What Must Be True

The spec explorer includes a branch filter dropdown in the tree panel header that allows users to filter the displayed assertions by branch.

## Success Criteria

- ✅ Dropdown appears in tree panel header (below project name)
- ✅ Options: "All branches" + all unique branch names found in assertions
- ✅ Default selection: "All branches"
- ✅ Selecting a branch filters tree to show only:
  - Assertions on that branch
  - Assertions with no branch field (default to main)
  - Parent specs that have at least one matching assertion
- ✅ Filter updates tree display without page reload (JavaScript)
- ✅ Specs with no matching assertions are hidden
- ✅ Counter updates: "X specs, Y assertions (filtered to branch: Z)"

## Example

```
┌─────────────────────────────┐
│ Spec Tree - spekk-cli       │
│ 5 specs, 12 assertions      │
│                             │
│ Branch: [All branches ▾]    │
└─────────────────────────────┘
```

When "feature/chat-system" selected:
```
┌─────────────────────────────┐
│ Spec Tree - spekk-cli       │
│ 1 spec, 4 assertions        │
│ (filtered to: chat-system)  │
│                             │
│ Branch: [feature/chat-sys ▾]│
└─────────────────────────────┘
```

## Implementation Notes

- Collect unique branch names from assertions during HTML generation
- Add `<select>` dropdown with onChange handler
- JavaScript filters by toggling `display: none` on tree items
- Preserve existing expand/collapse state during filtering
