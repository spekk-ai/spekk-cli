---
id: agent-spawns-use-inherited-stdio
parent: spekk-cli
created: 2026-03-16T00:00:00Z
priority: 1
status: done
---

# CLI agent spawns use inherited stdio

All CLI commands that launch Claude Code (`coach`, `builder`, `observer`, loops) pass the activation message as a positional argument with `stdio: 'inherit'`. No Claude spawn site pipes to stdin.

This ensures cross-platform compatibility — on Windows, Claude Code uses Ink which requires a real TTY for raw mode. Piped stdin breaks with "Raw mode is not supported".

## Success Criteria

- No Claude spawn in `src/coach/cli.js`, `src/builder/cli.js`, `src/observer/cli.js`, or `src/loops/index.js` uses `stdio: ['pipe', 'inherit', 'inherit']`
- All Claude spawns use `stdio: 'inherit'` and pass the message as a positional arg
- No `stdin.write()` / `stdin.end()` / EPIPE handling exists for Claude spawns
- `src/serve/index.js` is unchanged (intentionally pipes for WebSocket protocol)
- `buildClaudeSpawnConfig()` interactive branch is unchanged (already uses `stdio: 'inherit'`)
