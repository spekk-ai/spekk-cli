---
id: node-delegates-to-go
parent: golang-parser
created: 2026-04-05T12:05:00Z
priority: 2
status: not_started
depends-on: go-parser-json-matches-node
branch: feature/golang-parser
---

# spekk next delegates to Go binary

The Node CLI detects a compiled Go binary and delegates parser commands to it, falling back to the JS parser if the binary is not found.

## Success Criteria

- Node CLI checks for Go binary at `./spekk` or `./bin/spekk-go` (platform-appropriate path)
- If Go binary exists and is executable, parser commands (`next`, with all flags) are delegated to it via child process
- Go binary's stdout is passed through as the CLI output
- Go binary's stderr (warnings) is passed through to stderr
- Exit code from Go binary is preserved
- If Go binary is not found, falls back to existing Node parser seamlessly
- No user-visible behavior change — same JSON output regardless of which parser runs
