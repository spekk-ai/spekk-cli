---
id: spawn-uses-positional-arg
parent: fix-windows-tty-spawn
created: 2026-03-16T12:00:00Z
priority: 1
status: done
---

# All Claude Spawns Pass Message as Positional Argument

## What Must Be True

Every site that spawns `claude` for an interactive agent session must pass the activation message as a positional argument in the args array, not via stdin piping.

The spawn call pattern must be:
```js
spawn('claude', ['--dangerously-skip-permissions', message], { stdio: 'inherit' })
```

NOT:
```js
spawn('claude', ['--dangerously-skip-permissions'], { stdio: ['pipe', 'inherit', 'inherit'] })
// followed by stdin.write(message) and stdin.end()
```

### Affected Spawn Sites

1. `src/coach/cli.js` - `launchCoachAgent()` function
2. `src/builder/cli.js` - `buildAssertion()` function
3. `src/builder/cli.js` - `buildClaudeSpawnConfig()` headless branch
4. `src/observer/cli.js` - `launchObserverAgent()` function
5. `src/loops/index.js` - `runBuilderLoop()` Claude spawn
6. `src/loops/index.js` - `runCoachLoop()` Claude spawn

### Important Constraints

- The `-p` flag must NOT be used (it activates non-interactive print mode)
- `src/serve/index.js` must NOT be changed (different protocol)
- The interactive branch of `buildClaudeSpawnConfig()` must NOT be changed (already correct)

## Success Criteria

- No Claude spawn site uses `stdio: ['pipe', 'inherit', 'inherit']` for interactive sessions
- No `stdin.write()` or `stdin.end()` calls exist for Claude processes
- No EPIPE error handling blocks for Claude stdin
- All spawn sites use `stdio: 'inherit'`
- The message is the last element in the spawn args array
- Existing tests pass with updated expectations
