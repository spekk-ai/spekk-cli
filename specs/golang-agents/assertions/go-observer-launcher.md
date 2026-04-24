---
id: go-observer-launcher
parent: golang-agents
created: 2026-04-05T12:17:00Z
priority: 1
status: done
depends-on: go-prompt-resolver
branch: feature/golang-migration
---

# Go observer agent launcher

The Go observer launcher resolves the observer prompt and spawns `claude` with inherited stdio.

## Success Criteria

- `spekk observer` launches claude with observer activation message
- `spekk observer --interval <seconds>` passes scan interval preference in activation message
- `spekk observer --quiet` passes quiet mode preference in activation message
- `spekk observer --help` displays help
- Claude process gets inherited stdio
- SIGINT forwarded to claude for graceful shutdown
- Exit code from claude preserved
