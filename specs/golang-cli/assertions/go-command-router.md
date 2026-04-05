---
id: go-command-router
parent: golang-cli
created: 2026-04-05T12:10:00Z
priority: 1
status: not_started
depends-on: go-project-structure
branch: feature/golang-cli
---

# Go binary routes all CLI commands

The Go binary at `cmd/spekk/main.go` is the single entry point that routes all `spekk <command>` invocations to the correct handler.

## Success Criteria

- `spekk` with no args runs the parser (default behavior)
- `spekk next` runs the parser with flags (`--all`, `--spec`, `--assertion`, `--all-branches`)
- `spekk coach`, `spekk builder`, `spekk observer` route to agent launchers
- `spekk loop builder`, `spekk loop coach` route to loop handlers
- `spekk status` routes to status display
- `spekk show` routes to show handler (with `--watch` flag)
- `spekk serve` routes to serve handler (with `--port`, `--host` flags)
- `spekk sandbox <subcommand>` routes to sandbox handler
- `spekk help`, `spekk --help`, `spekk -h` display help text with available commands and skills
- Unknown commands print error and exit with code 1
