---
id: tmp-directory-established
parent: cleanup-repository
created: 2026-01-28T21:20:00Z
priority: 1
status: done
---

# Tmp Directory Established

## What Must Be True

A `.tmp/` directory exists as the centralized location for all temporary development artifacts.

## Implementation

1. Create `.tmp/` directory in repository root
2. Move existing temporary files to `.tmp/`:
   - `debug-coach.js` → `.tmp/debug-coach.js`
   - `debug-copy.js` → `.tmp/debug-copy.js`
   - `test-fix.js` → `.tmp/test-fix.js`
   - `debug-mock-mowlIc/` → `.tmp/debug-mock-mowlIc/`
   - `debug-test/` → `.tmp/debug-test/`
   - `temp-test-DB4WyD/` → `.tmp/temp-test-DB4WyD/`
   - `temp-test-debug/` → `.tmp/temp-test-debug/`
   - `temp-test-mKLbyt/` → `.tmp/temp-test-mKLbyt/`
   - `spekk-app-reference/` → `.tmp/spekk-app-reference/`
   - `app/` → `.tmp/app/` (incomplete implementation artifacts)

## Success Criteria

- ✅ `.tmp/` directory exists
- ✅ All temporary artifacts moved to `.tmp/`
- ✅ Repository root is clean of temporary files
- ✅ All functionality still works after moving files

## Convention

Going forward, builders should place all debug files, test artifacts, and temporary outputs in `.tmp/` rather than the repository root.