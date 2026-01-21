---
id: no-implementation-outside-app
parent: spec-parser
created: 2026-01-20T18:50:00Z
priority: 1
status: done
---

# All Code Lives in src/

## What Must Be True

**All implementation code lives under the `src/` directory.**

The project uses standard CLI structure with `src/` for implementation and `bin/` for CLI entry points.

## Directory Structure Required

```
src/
├── parser/          # Parser implementation
│   ├── index.js     # Core logic
│   ├── cli.js       # CLI interface
│   └── __tests__/   # Tests (optional)
├── coach/           # Coach implementation
│   ├── index.js
│   └── cli.js
└── builder/         # Builder implementation
    ├── index.js
    └── cli.js

bin/
└── spekk.js        # Main CLI entry point
```

## CLI Entry Points

**Package.json scripts:**
```json
{
  "scripts": {
    "next": "node src/parser/cli.js",
    "coach": "node src/coach/cli.js",
    "builder": "node src/builder/cli.js"
  },
  "bin": {
    "spekk": "./bin/spekk.js"
  }
}
```

**Direct invocation:**
```bash
node src/parser/cli.js
# or
./bin/spekk.js next
```

## Tests Location

Tests should live co-located with code using `__tests__/` convention:
- `src/parser/__tests__/parser.test.js`
- `src/**/__tests__/*.test.js`

## Success Criteria

- ✅ All implementation code lives in `src/`
- ✅ CLI entry point exists in `bin/spekk.js`
- ✅ Module CLIs exist as `src/*/cli.js` files
- ✅ Parser works via `npm run next`
- ✅ Main CLI works via `spekk` command
- ✅ No code scattered in random locations
