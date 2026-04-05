---
id: node-parser-removed
parent: golang-parser
created: 2026-04-05T12:06:00Z
priority: 3
status: not_started
depends-on: node-delegates-to-go
branch: feature/golang-parser
---

# Node.js parser code removed

After the Go parser is validated and the delegation layer is stable, the Node.js parser implementation is deleted.

## Success Criteria

- `src/parser/index.js` is deleted
- `src/parser/cli.js` is replaced with a thin wrapper that calls the Go binary (no fallback)
- All `src/parser/__tests__/*.test.js` files are deleted
- Equivalent test coverage exists as Go tests in `internal/parser/`
- `npm run next` invokes the Go binary directly
- `go test ./internal/parser/...` passes all tests
- No JS parser logic remains anywhere in the codebase
