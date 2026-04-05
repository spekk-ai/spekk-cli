---
id: go-project-structure
parent: golang-parser
created: 2026-04-05T12:00:00Z
priority: 1
status: not_started
branch: feature/golang-migration
---

# Go module and project structure exists

A Go module is initialized at the project root with standard Go project layout.

## Success Criteria

- `go.mod` exists at project root with module path `github.com/spekk-dev/spekk-cli`
- `cmd/spekk/main.go` exists as the CLI entry point
- `internal/parser/` package exists for parser logic
- `go build ./cmd/spekk` compiles without errors
- Binary accepts `next` subcommand (can be a stub that exits 0)
