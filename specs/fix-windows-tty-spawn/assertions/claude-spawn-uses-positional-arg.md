---
id: claude-spawn-uses-positional-arg
parent: fix-windows-tty-spawn
created: 2026-03-16T00:00:00Z
priority: 1
status: not_started
---

# Claude spawn sites pass prompt as positional argument with inherited stdio

All Claude Code spawn sites use `stdio: 'inherit'` and pass the activation message as a positional CLI argument rather than piping to stdin. No spawn site for Claude uses `stdio: ['pipe', 'inherit', 'inherit']`.

## Success Criteria

- `src/coach/cli.js`, `src/builder/cli.js`, `src/observer/cli.js`, and `src/loops/index.js` all spawn Claude with `stdio: 'inherit'`
- The activation message is a positional argument in the spawn args array, not written to stdin
- No `stdin.write()` / `stdin.end()` / EPIPE handling exists for Claude spawns
- `src/serve/index.js` is unchanged (intentionally uses pipe for WebSocket protocol)
