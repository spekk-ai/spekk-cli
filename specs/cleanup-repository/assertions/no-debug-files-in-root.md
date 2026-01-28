---
id: no-debug-files-in-root
parent: cleanup-repository
created: 2026-01-28T21:18:00Z
priority: 1
status: not_started
---

# No Debug Files in Root

## What Must Be True

No debug or test files exist in the repository root directory.

## Files That Must Not Exist

```
debug-coach.js
debug-copy.js
test-fix.js
debug-mock-mowlIc/
debug-test/
temp-test-DB4WyD/
temp-test-debug/
temp-test-mKLbyt/
```

## Success Criteria

- ✅ No `debug-*.js` files in root directory
- ✅ No `test-*.js` files in root directory (except proper test files in test directories)
- ✅ No `debug-*` directories in root
- ✅ No `temp-test-*` directories in root
- ✅ Repository root is clean of temporary development artifacts

## Implementation Notes

These files should be:
1. Deleted if they're truly temporary
2. Moved to appropriate directories if they're needed
3. Added to `.gitignore` if they're generated during development