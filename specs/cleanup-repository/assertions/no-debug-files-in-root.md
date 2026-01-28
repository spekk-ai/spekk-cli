---
id: no-debug-files-in-root
parent: cleanup-repository
created: 2026-01-28T21:18:00Z
priority: 1
status: done
---

# No Debug Files in Root

## What Must Be True

No debug or test files exist in the repository root directory. All temporary artifacts must be in `.tmp/`.

## Files That Must Not Exist in Root

```
debug-coach.js
debug-copy.js
test-fix.js
debug-mock-mowlIc/
debug-test/
temp-test-DB4WyD/
temp-test-debug/
temp-test-mKLbyt/
observations/
```

## Success Criteria

- ✅ Repository root contains no temporary development artifacts
- ✅ All debug/test files are moved to `.tmp/` directory
- ✅ Repository follows clean organization principles

## Implementation Notes

These files should be moved to `.tmp/` rather than deleted, in case they're still needed for development or debugging purposes.