---
id: show-convergence-terminal
parent: spec-explorer-web-interface
created: 2026-02-25T19:21:00Z
priority: 2
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Show Convergence Terminal

## What Must Be True

Terminal assertions (no children) have a small, subtle endpoint dot that indicates chain completion status. Done nodes are visually quiet — they mark endpoints without drawing attention away from the assertion stations themselves.

## Success Criteria

- Each terminal assertion gets its own Done endpoint node
- Done nodes are small: r=5 (compared to r=8 for assertion stations)
- No text label underneath Done nodes (no "Done" text, no "✓ Complete")
- Done node fill is green (#10b981) when the entire upstream chain is done
- Done node fill is gray (#94a3b8) when any upstream assertion is not done
- Done nodes have white stroke border (stroke-width: 2) for visibility
- Done nodes are non-interactive (no click handler, `pointer-events: none`)
- If branch has only one terminal assertion, no Done endpoint needed
- Convergence line from terminal assertion to its Done node uses same gray (#94a3b8) at low opacity

## Visual Example

**Done nodes are subtle endpoints:**
```
○───○───○───●    (● = small green dot, whole chain done)
     └──○───◦    (◦ = small gray dot, chain not fully done)
```

## When to Show Done Nodes

**Show when:**
- Branch has 2+ terminal assertions (parallel endpoints)

**Don't show when:**
- Branch has 1 terminal assertion (already clear endpoint)

## Implementation Notes

- Detect terminal assertions: assertions with no children depending on them
- Done node position: `x = terminalPos.x + 120`, `y = terminalPos.y`
- Check upstream chain: walk `dependsOn` links back to root, check if all are `status: done`
- `allDone ? '#10b981' : '#94a3b8'` for fill color
- No `<text>` elements on Done nodes
- CSS: `.metro-terminus { cursor: default; pointer-events: none; }`
