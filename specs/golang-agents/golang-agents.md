---
id: golang-agents
created: 2026-04-05T12:15:00Z
priority: 1
---

# Go Agent Launchers

Port the coach, builder, and observer agent launchers from Node.js to Go.

These are thin wrappers — they resolve prompts, optionally activate skills, then exec `claude` as a subprocess with inherited stdio. The real logic lives in the prompt/skill resolvers (ported in golang-cli).

## Current Architecture

- `src/coach/cli.js` — resolve prompt, optionally activate skill, spawn claude (162 lines)
- `src/builder/cli.js` — resolve prompt, parse flags, spawn claude in various modes (574 lines)
- `src/observer/cli.js` — resolve prompt, spawn claude (160 lines)
- `src/loops/index.js` — builder/coach orchestration loops (293 lines)

## Strategy

- Each agent launcher becomes a Go function in `internal/agents/`
- All use the shared prompt resolver and skill resolver from `internal/cli/`
- Builder is the most complex (multiple modes: once, continuous, interactive, dry-run, confirm)
- Loops become Go functions that call the agent launchers in a loop
