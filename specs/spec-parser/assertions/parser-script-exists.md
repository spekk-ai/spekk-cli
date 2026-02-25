---
id: parser-script-exists
parent: spec-parser
created: 2026-01-20T16:05:00Z
priority: 1
status: done
branch: feature/spec-parser
---

# Parser Module Must Exist and Work

## What Must Be True

A working **Node.js parser module** must exist in `src/parser/` that:
1. Reads spec folders from `specs/` directory
2. Parses frontmatter from spec and assertion files
3. Validates the format according to declared rules
4. Identifies the next highest-priority incomplete assertion
5. Outputs valid JSON to stdout

## File Location

```
src/
└── parser/
    ├── index.js     # Core implementation
    ├── cli.js       # CLI entry point
    └── __tests__/   # Tests (optional)
```

## Expected Behavior

When executed via npm script:
```bash
npm run next
```

Or directly:
```bash
node src/parser/cli.js
```

Should output JSON:
```json
{
  "type": "assertion",
  "id": "enforces-folder-structure",
  "parent": "spec-parser",
  "file": "specs/spec-parser/assertions/enforces-folder-structure.md",
  "priority": 0,
  "status": "not_started",
  "title": "Parser Must Enforce Folder Structure",
  "content": "... full markdown content ..."
}
```

## Required Dependencies

- Node.js runtime with ES modules support
- Built-in YAML frontmatter parsing (no external dependencies required)
- Should work with minimal dependencies

## Success Criteria

This assertion is "done" when:
- ✅ `src/parser/index.js` exists with core implementation
- ✅ `src/parser/cli.js` exists with CLI entry point
- ✅ Parser is executable via `npm run next`
- ✅ Parser reads folder structure from `specs/`
- ✅ Parser parses YAML frontmatter correctly
- ✅ Parser identifies next priority assertion
- ✅ Parser outputs valid JSON
- ✅ Parser handles errors gracefully (malformed files, missing fields, etc.)
- ✅ Parser runs in < 100ms for typical spec count

## Test Cases

The parser should handle:
1. Empty `specs/` directory → output indicating no work to do
2. All specs `done` → output indicating all complete
3. Multiple specs with different priorities → return highest priority
4. Malformed YAML → clear error message
5. Missing required fields → validation error
6. Duplicate IDs → detection and error

## Implementation Notes

This is the bootstrap: we need this module to exist before we can be truly spec-driven. Once it works, the ralph-loop can use it to drive all subsequent work.
