---
id: golang-cli
created: 2026-04-05T12:10:00Z
priority: 1
---

# Go CLI Core

Port the CLI entry point, command routing, flag parsing, prompt resolution, and skill resolution from Node.js to Go.

This is the spine of the binary — it routes `spekk <command>` to the correct handler. Depends on the Go parser being functional first, since the default command runs the parser.

## Current Architecture

- `bin/spekk.js` — switch/case command router
- `src/cli/parse-flags.js` — generic flag parser (53 lines)
- `src/cli/prompt-resolver.js` — layered prompt resolution (base → global → local) (165 lines)
- `src/cli/skill-resolver.js` — layered skill discovery (local → global → package) (159 lines)

## Strategy

- Go binary becomes the single `spekk` entry point
- All command routing moves to Go
- Prompt and skill resolution ported as Go packages
- Agent commands exec `claude` as a subprocess (same pattern, different language)
