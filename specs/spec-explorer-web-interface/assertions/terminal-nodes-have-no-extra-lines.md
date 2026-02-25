---
id: fix-erroneous-terminal-lines
parent: spec-explorer-web-interface
created: 2026-02-25T20:32:00Z
priority: 2
status: done
branch: feature/dependency-visualization
---

# Fix Erroneous Terminal Lines

## What Must Be True

No erroneous lines appear to the right of topmost terminal nodes in the metro map.

## Success Criteria

- ✅ Terminal nodes have no extra lines extending to their right
- ✅ Track lines only connect assertions with actual dependencies
- ✅ "Done" convergence terminal is properly connected
- ✅ Visual is clean with no stray lines

## Problem Description

User reports: "There's an erroneous extra line to the right of the topmost terminals - not sure why"

This likely means:
- Terminal assertions (those with no children) have lines extending beyond them
- The "Done" convergence terminal might have extra connection lines
- Track generation logic is drawing lines when it shouldn't

## Investigation Steps

1. Check track line generation in `generateMetroMap()` function
2. Look for line drawing between:
   - Terminal assertions and non-existent children
   - Convergence terminal and assertions
3. Check if "Done" terminal connector logic has bugs
4. Verify `dependsOn` relationships are correctly identified

## Likely Causes

**Cause A: Track line extends beyond terminal**
```javascript
// Bug: Drawing line even when no child exists
const lineX2 = assertionX + spacing; // Wrong: extends past terminal
```

**Cause B: Convergence terminal connection issue**
```javascript
// Bug: Drawing lines TO convergence terminal incorrectly
terminals.forEach(terminal => {
  drawLine(terminal.x, convergenceX); // May be drawing extra line
});
```

**Cause C: Done terminal appears when it shouldn't**
```javascript
// Bug: Showing convergence when only 1 terminal exists
if (terminals.length >= 1) { // Should be > 1
  showConvergenceTerminal();
}
```

## Solution Approach

1. **Identify terminal assertions:**
   ```javascript
   const terminals = assertions.filter(a =>
     !assertions.some(child => child.dependsOn === a.id)
   );
   ```

2. **Don't draw lines beyond terminals:**
   ```javascript
   // Only draw lines between assertions with dependencies
   assertions.forEach(assertion => {
     if (assertion.dependsOn) {
       const parent = findAssertion(assertion.dependsOn);
       drawLine(parent.x, parent.y, assertion.x, assertion.y);
     }
   });
   ```

3. **Check convergence terminal logic:**
   ```javascript
   // Only show if multiple terminals exist
   if (terminals.length > 1) {
     showConvergenceTerminal();
     terminals.forEach(t => drawLine(t, convergence));
   }
   ```

## Implementation Notes

- Review SVG generation carefully
- Check both track lines and convergence connectors
- May need to debug by adding console.logs to track generation
- Consider visual debugging: color code different line types
