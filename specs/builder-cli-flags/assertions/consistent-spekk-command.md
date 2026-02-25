---
id: consistent-spekk-command
parent: builder-cli-flags
created: 2026-02-25T00:14:00Z
priority: 1
status: done
---

# Consistent Use of Local spekk Command

## Description

`buildSpekkNextCommand()` hardcodes the string `'spekk next'` for injection into Claude's prompt, but `getNextAssertion()` correctly uses `getSpekkCommand()` which resolves to the local `bin/spekk.js`. This inconsistency means Claude may use a globally installed `spekk` instead of the local one, causing version mismatches.

## Success Criteria

- `buildSpekkNextCommand()` uses `getSpekkCommand()` instead of hardcoding `'spekk next'`
- All references to the spekk CLI within the codebase use `getSpekkCommand()` for consistency
- Claude's injected prompt references the local `bin/spekk.js` path, not the global `spekk`
- No hardcoded `'spekk'` command strings exist outside of `getSpekkCommand()`

**Tests:** src/builder/__tests__/cli.test.js (buildSpekkNextCommand consistency suite)
