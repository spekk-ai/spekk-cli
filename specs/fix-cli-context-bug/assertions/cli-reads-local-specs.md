---
id: cli-reads-local-specs
parent: fix-cli-context-bug
created: 2026-01-28T21:22:00Z
priority: 1
status: done
---

# CLI Reads Local Specs

## What Must Be True

The `spekk next` command reads specs from the current working directory's `specs/` folder, not from the CLI installation directory.

## Current Bug

The CLI is reading from its own installation `specs/` directory instead of the user's current project directory.

## Success Criteria

- ✅ `spekk next` reads from `process.cwd() + '/specs/'`
- ✅ When run from different directories, CLI operates on those directories' specs
- ✅ CLI doesn't read from spekk-cli installation directory when run externally
- ✅ Error handling when no `specs/` directory exists in current working directory

## Test Cases

1. **From spekk-cli directory:** Should read `./specs/`
2. **From external project:** Should read that project's `./specs/`
3. **From directory with no specs:** Should show appropriate error message

## Implementation Notes

The parser logic needs to:
- Use `process.cwd()` to determine the working directory
- Look for specs in `${process.cwd()}/specs/`
- Not hardcode paths to the CLI's own specs directory

## Tests

**Tests:** src/parser/__tests__/cli-reads-local-specs.test.js

## Validation Commands

```bash
# Test from different directories
cd /path/to/external/project && spekk next
cd /another/project && spekk next
```

Each should read specs from their respective local directories.