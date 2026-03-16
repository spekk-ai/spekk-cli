---
id: parser-skips-specs-without-assertions-dir
parent: robust-error-handling
created: 2026-03-16T18:00:00Z
priority: 1
status: not_started
---

# Parser Skips Specs Without Assertions Directory

**Closes:** #18, #30, #41

## What Must Be True

When a spec directory exists with a valid parent spec file but no `assertions/` subdirectory, the parser warns to stderr and skips that spec — it never throws or crashes.

## Success Criteria

- `validateFolderStructure` does not throw when `assertions/` is missing; it logs a warning to stderr and continues
- `spekk next` returns valid JSON even when some spec directories lack `assertions/`
- `spekk show` renders all valid specs and skips incomplete ones without crashing
- `spekk status` counts only parseable specs; incomplete ones are excluded with a warning
- A spec directory with a parent `.md` but no `assertions/` is treated as if it has zero assertions (not as an error)
