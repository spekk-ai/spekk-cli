---
id: parser-runs-without-syntax-errors
parent: fix-parser-syntax-error
created: 2026-01-28T21:15:00Z
priority: 1
status: not_started
---

# Parser Runs Without Syntax Errors

## What Must Be True

The parser at `src/parser/index.js` must execute without any syntax errors when the spekk CLI is invoked.

## Current Issue

Line 78 contains an illegal `continue` statement that's not inside a loop:
```javascript
continue;  // This is outside any loop - syntax error!
```

## Success Criteria

- ✅ `spekk` command executes without syntax errors
- ✅ `spekk next` command executes without syntax errors
- ✅ `spekk builder` command executes without syntax errors
- ✅ No "Illegal continue statement" errors
- ✅ Parser successfully parses spec files when no syntax errors exist

## Implementation Notes

The fix likely involves either:
1. Removing the orphaned `continue` statement if it's not needed
2. Wrapping the logic in a proper loop if iteration was intended
3. Replacing `continue` with `return` if early exit was intended
4. Restructuring the code to properly handle the control flow

## Validation

Run these commands without syntax errors:
```bash
spekk
spekk next
spekk status
```