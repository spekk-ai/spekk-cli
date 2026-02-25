---
id: show-convergence-terminal
parent: spec-explorer-web-interface
created: 2026-02-25T19:21:00Z
priority: 2
status: done
branch: feature/dependency-visualization
---

# Show Convergence Terminal

## What Must Be True

The metro map shows a final "done" terminal station where parallel workstreams converge, making it clear how parallel work relates to completion.

## Success Criteria

- ✅ If branch has multiple terminal assertions (no children), add a visual "Done" terminus
- ✅ Terminal assertions connect to the "Done" terminus with track lines
- ✅ "Done" terminus styled distinctly (e.g., larger dot, different color, checkmark icon)
- ✅ If branch has only one terminal assertion, no "Done" terminus needed (already clear)
- ✅ Label shows "Done" or "✓ Complete" below terminus station
- ✅ Clicking "Done" terminus shows branch summary or does nothing (not a real assertion)

## Visual Example

**Before (unclear convergence):**
```
A ──── B ──── C
       └──── D
```
Are C and D both endpoints? Unclear.

**After (clear convergence):**
```
A ──── B ──── C ─────┐
       └──── D ──────┤──── ✓ Done
```
Both C and D converge to completion.

## When to Show Terminus

**Show terminus when:**
- Branch has 2+ terminal assertions (parallel endpoints)
- Makes convergence clear

**Don't show terminus when:**
- Branch has 1 terminal assertion (already clear endpoint)
- All assertions in branch are done (redundant)

## Implementation Notes

- Detect terminal assertions: `assertions.filter(a => !assertions.some(child => child.dependsOn === a.id))`
- If `terminalAssertions.length > 1`, add virtual "Done" node
- Position "Done" node at `x = maxX + spacing`
- Draw curved connectors from each terminal to "Done"
- Style: larger circle (12px radius), success color (#10b981), checkmark icon
- Make non-interactive (no click handler, or show branch summary)
