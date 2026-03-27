---
id: target-directory-structure
parent: spec-parser
created: 2026-01-21T19:15:00Z
priority: 1
status: done
branch: feature/spec-parser
---

# CLI Directory Structure

## What Must Be True

**The project must follow standard Node.js CLI structure:**

```
spekk-cli/
├── src/                  # All implementation code
│   ├── parser/          # Spec parser
│   │   ├── index.js     # Core logic
│   │   ├── cli.js       # CLI interface
│   │   └── __tests__/   # Tests (optional)
│   ├── coach/           # Coach agent
│   │   ├── index.js
│   │   └── cli.js
│   └── builder/         # Builder agent
│       ├── index.js
│       └── cli.js
├── bin/                 # CLI entry points
│   └── spekk.js        # Main executable
├── specs/               # Specifications
├── package.json         # Package config with "bin" field
└── ...
```

## Directory Responsibilities

**src/** - All implementation code lives here
- Organized by module (parser, coach, builder)
- Each module has index.js (logic) and cli.js (interface)
- Tests co-located with code in __tests__/

**bin/** - CLI executables
- Main entry point for the `spekk` command
- Thin wrapper that imports from src/

**specs/** - Specifications only
- No code, only markdown spec files
- Follows spec-parser folder structure rules

## npm Package Configuration

**package.json must include:**
```json
{
  "type": "module",
  "bin": {
    "spekk": "./bin/spekk.js"
  },
  "scripts": {
    "next": "node src/parser/cli.js",
    "coach": "node src/coach/cli.js",
    "builder": "node src/builder/cli.js"
  }
}
```

## What Must NOT Exist

- ❌ `app/` directory (wrong - that's for spekk-app, not spekk-cli)
- ❌ `spec-parser/` as standalone at root (parser is part of CLI, lives in src/)
- ❌ Code scattered outside src/ directory
- ❌ Django or Python code (this is a Node.js CLI)

## Success Criteria

This assertion is "done" when:

- ✅ `src/` directory exists with all implementation code
- ✅ `src/parser/`, `src/coach/`, `src/builder/` exist
- ✅ `bin/spekk.js` exists as main CLI entry point
- ✅ `npm run next` executes parser correctly
- ✅ `node src/parser/cli.js` works directly
- ✅ Parser outputs valid JSON with correct format
- ✅ No `app/` directory exists
- ✅ No standalone `spec-parser/` at root
- ✅ Clean separation: src/ for code, bin/ for CLI, specs/ for specs

## Rationale

This is standard Node.js CLI structure:
- **src/** is convention for source code
- **bin/** is convention for executables
- Keeps the CLI portable and publishable to npm
- Clear separation of concerns
