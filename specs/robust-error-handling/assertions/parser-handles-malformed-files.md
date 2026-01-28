---
id: parser-handles-malformed-files
parent: robust-error-handling
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---

# Parser Handles Malformed Files

## What Must Be True

The parser gracefully handles malformed files by warning and skipping them, never crashing the entire CLI.

## Current Problem

Parser crashes with error like:
```
Field 'affected_specs' must be an array in observations/file.md
```

## Success Criteria

- ✅ Parser logs warnings for malformed files to stderr
- ✅ Parser skips malformed files and continues processing
- ✅ CLI returns valid JSON even when some files are malformed  
- ✅ `spekk next` works even with malformed observation files
- ✅ Builder can continue to operate despite file parsing issues

## Implementation

Wrap file parsing in try/catch blocks:
```javascript
try {
  // parse file
} catch (error) {
  console.warn(`Warning: Skipping malformed file ${filePath}: ${error.message}`);
  continue; // skip this file
}
```