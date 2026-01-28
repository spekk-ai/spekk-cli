---
id: cleanup-repository
created: 2026-01-28T21:18:00Z
priority: 1
status: not_started
---

# Clean Up Repository

## Overview

The repository root directory contains temporary files, debug scripts, and test artifacts that should not be committed to version control. These files were likely created during development and testing but should be removed.

## Files to Clean Up

**Debug Files:**
- `debug-coach.js`
- `debug-copy.js` 
- `test-fix.js`

**Debug Directories:**
- `debug-mock-mowlIc/`
- `debug-test/`

**Temporary Test Directories:**
- `temp-test-DB4WyD/`
- `temp-test-debug/`
- `temp-test-mKLbyt/`

**Observation Files:**
- `observations/` directory contains hundreds of timestamped markdown files

## Repository Hygiene Principles

- Temporary files and directories should never be committed
- Debug scripts should be in `.gitignore` or removed when done
- Test artifacts should be cleaned up after test runs
- Generated files that aren't source of truth should be excluded