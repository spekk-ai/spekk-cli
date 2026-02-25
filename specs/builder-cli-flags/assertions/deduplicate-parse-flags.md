---
id: deduplicate-parse-flags
parent: builder-cli-flags
created: 2026-02-25T00:14:00Z
priority: 1
status: done
---

# Deduplicate parseFlags Into Shared Utility

## Description

Two separate `parseFlags` functions exist — one in `src/builder/cli.js` and another in `src/parser/cli.js`. These should be consolidated into a single shared utility.

## Success Criteria

- One `parseFlags` function exists as a shared utility (not duplicated across modules)
- Both `src/builder/cli.js` and `src/parser/cli.js` import from the shared utility
- No duplicate flag-parsing logic across the codebase
- Existing tests continue to pass after consolidation

**Tests:** src/cli/__tests__/parse-flags.test.js
