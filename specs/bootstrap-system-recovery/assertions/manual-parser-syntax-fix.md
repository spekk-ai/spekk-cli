---
id: manual-parser-syntax-fix
parent: bootstrap-system-recovery
created: 2026-01-28T21:30:00Z
priority: 1
status: done
---

# Manual Parser Syntax Fix

## What Must Be True

The parser at `src/parser/index.js` line 78 has the illegal `continue` statement fixed so `spekk next` can execute.

## Critical Bootstrap Fix

**Location:** `src/parser/index.js:78`
**Issue:** `continue;` statement outside of loop context
**Error:** `SyntaxError: Illegal continue statement: no surrounding iteration statement`

## Manual Fix Required

Since the builder cannot run to fix this automatically, this requires direct code editing:

1. **Examine line 78** in `src/parser/index.js`
2. **Remove or replace** the illegal `continue` statement
3. **Likely solutions:**
   - Remove the line if it's not needed
   - Replace with `return` if early exit is intended
   - Wrap in proper loop if iteration was intended

## Success Criteria

- ✅ `spekk next` executes without syntax errors
- ✅ `node src/parser/cli.js` runs successfully
- ✅ Parser can be called by builder loop

## Urgency

**BLOCKING:** This must be fixed before any other development can proceed. The entire spec-driven system depends on the parser working.