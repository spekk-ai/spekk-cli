---
id: builder-uses-npm-test-commands
parent: fix-builder-test-validation
created: 2026-01-28T21:40:00Z
priority: 1
status: done
---

# Builder Uses NPM Test Commands

## What Must Be True

The builder agent prompt instructs builders to use the correct test commands for this project (`npm test`) instead of non-existent `just` commands.

## Current Problem

Builder prompt at line ~82-85 says:
```bash
# Use justfile commands for standardized testing
just test           # Run all tests (server + client)
just server-test    # Run server tests only  
just client-test    # Run client tests only
```

But this project doesn't use `just` - it uses npm scripts.

## Success Criteria

- ✅ Builder prompt references `npm test` instead of `just test`
- ✅ Remove references to non-existent `just` commands
- ✅ Instructions match actual project test setup
- ✅ Builder validates tests actually pass before marking `done`

## Correct Commands

Should be:
```bash
npm test              # Run all tests 
npm run test:impl     # Run implementation tests only
npm run test:specs    # Run spec validation tests only
```