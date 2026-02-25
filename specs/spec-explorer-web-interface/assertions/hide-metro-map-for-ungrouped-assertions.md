---
id: hide-metro-map-for-ungrouped-assertions
parent: spec-explorer-web-interface
created: 2026-02-25T20:01:00Z
priority: 1
status: not_started
branch: feature/dependency-visualization
---

# Hide Metro Map for Ungrouped Assertions

## What Must Be True

When an assertion has no `branch` field (defaults to main) and there are many unrelated main assertions, the metro map is hidden and replaced with a notice explaining why.

## Success Criteria

- ✅ If assertion has no branch field (or `branch: main`), check if branch is meaningful
- ✅ A branch is "meaningful" if it has dependencies between its assertions
- ✅ If branch has no dependencies (all isolated assertions), hide metro map
- ✅ Show notice: "ℹ️ No branch dependencies to visualize. This assertion is on the main branch with no related dependencies."
- ✅ If assertion has `branch: feature/xyz`, always show metro map (even without deps)
- ✅ Notice is styled distinctly (light blue background, info icon)

## Rationale

Main branch often has many unrelated completed specs. Showing all of them in the metro map creates a useless horizontal line of dots with no connections. Better to hide it and explain why.

## Logic

```javascript
function shouldShowMetroMap(assertion, branchAssertions) {
  // Always show for feature branches
  if (assertion.branch && assertion.branch !== 'main') {
    return true;
  }

  // For main branch, only show if there are dependencies
  const hasDependencies = branchAssertions.some(a => a.dependsOn);
  return hasDependencies;
}
```

## Visual Example

**Notice instead of metro map:**
```
╔══════════════════════════════════╗
║ My Assertion                     ║
║ Status: done  Priority: 1        ║
║ Branch: main                     ║
╠══════════════════════════════════╣
║ ℹ️ No branch dependencies       ║
║                                  ║
║ This assertion is on the main    ║
║ branch with no related           ║
║ dependencies. Branch dependencies║
║ are shown for feature branches   ║
║ and main branch assertions with  ║
║ dependency chains.               ║
╠══════════════════════════════════╣
║ ## What Must Be True             ║
║ [assertion content...]           ║
╚══════════════════════════════════╝
```

## Implementation Notes

- Check `shouldShowMetroMap()` before generating metro map SVG
- If false, generate notice HTML instead
- Notice should be visually distinct but not alarming (info, not warning)
- Consider adding link to docs about branch organization
