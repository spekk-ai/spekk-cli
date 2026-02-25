---
id: interactive-flag
parent: builder-cli-flags
created: 2026-02-25T00:14:00Z
priority: 2
status: done
---

# --interactive Flag Runs Claude in Non-Headless Mode

## Description

The `--interactive` / `-i` flag launches Claude in interactive (non-headless) mode, allowing the user to collaborate with Claude during the build rather than running fully autonomously.

**Tests:** `src/builder/__tests__/cli.test.js` (buildClaudeSpawnConfig suite)

## Success Criteria

- `spekk builder --interactive` or `spekk builder -i` launches Claude without `--print` flag
- User can interact with Claude during the build session
- Default (no flag) remains headless/autonomous mode
- Combines with `--assertion` to interactively build a specific assertion
- Combines with `--spec` to interactively build within a spec's scope
