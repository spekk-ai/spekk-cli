---
id: go-flag-parser
parent: golang-cli
created: 2026-04-05T12:11:00Z
priority: 1
status: done
depends-on: go-command-router
branch: feature/golang-cli
---

# Go flag parser handles all CLI flags

A shared flag parsing utility in Go that handles boolean and string flags for all commands.

## Success Criteria

- Parses boolean flags (e.g., `--once`, `--dry-run`, `--watch`, `--confirm`, `--interactive`)
- Parses string flags with values (e.g., `--spec auth`, `-s auth`, `--assertion foo`)
- Supports short aliases (e.g., `-s` for `--spec`, `-d` for `--dry-run`, `-c` for `--confirm`)
- Applies defaults (false for boolean, empty for string)
- Unknown flags are ignored (not fatal)
- Each command defines its own flag set using the shared utility
