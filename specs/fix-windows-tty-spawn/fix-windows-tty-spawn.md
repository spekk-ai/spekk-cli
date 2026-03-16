---
id: fix-windows-tty-spawn
created: 2026-03-16T12:00:00Z
priority: 1
---

# Fix Windows TTY Compatibility for Claude Code Spawning

## Problem

Spekk spawns Claude Code with `stdio: ['pipe', 'inherit', 'inherit']` and writes the activation message to stdin. On Windows, Claude Code uses Ink which requires a real TTY for raw mode. When stdin is piped, Ink crashes with "Raw mode is not supported".

This affects all Windows users and blocks cross-platform adoption.

## What Must Be True

All Claude Code spawn sites must pass the prompt message as a CLI positional argument instead of piping to stdin, and use `stdio: 'inherit'` so Claude receives a real TTY on all platforms.

### Affected Files

1. `src/coach/cli.js` - coach agent spawn
2. `src/builder/cli.js` - builder agent spawn (main `buildAssertion` and `buildClaudeSpawnConfig` headless branch)
3. `src/observer/cli.js` - observer agent spawn
4. `src/loops/index.js` - builder loop and coach loop spawns

### Files NOT Modified

- `src/serve/index.js` - uses `-p` with stream-json for WebSocket protocol (intentionally non-interactive)
- The `interactive` branch of `buildClaudeSpawnConfig()` - already uses `stdio: 'inherit'`
- Any `execSync` with `['pipe', 'pipe', 'pipe']` - those are for git/parser, not Claude

### Approach

For each spawn site:
1. Move the message into the spawn args array as the last positional argument
2. Change `stdio: ['pipe', 'inherit', 'inherit']` to `stdio: 'inherit'`
3. Remove the `stdin.write()`, `stdin.end()`, and EPIPE error handling blocks
4. Do NOT use the `-p` flag (that activates non-interactive print mode)
