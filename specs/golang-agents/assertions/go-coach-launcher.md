---
id: go-coach-launcher
parent: golang-agents
created: 2026-04-05T12:15:00Z
priority: 1
status: done
depends-on: go-prompt-resolver
branch: feature/golang-migration
---

# Go coach agent launcher

The Go coach launcher resolves the coach prompt, optionally activates a skill, and spawns `claude` with inherited stdio.

## Success Criteria

- `spekk coach` launches claude with the coach activation message
- `spekk coach <skill>` resolves the skill and appends skill content to the activation message
- `spekk coach meeting <file>` reads the transcript file and includes it in the activation message
- `spekk coach --help` displays help with available skills
- Claude process gets inherited stdio (real TTY)
- SIGINT forwarded to claude process for graceful shutdown
- Exit code from claude preserved
- Error message if `claude` CLI is not found (ENOENT)
