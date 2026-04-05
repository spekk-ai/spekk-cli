---
id: golang-parser
created: 2026-04-05T12:00:00Z
priority: 1
---

# Go Parser

Port the spec parser from Node.js to Go as the first phase of the full Go migration (strangler fig approach).

The parser is the foundation — every other module (builder, coach, show, status, loops) depends on its JSON output. Porting it first gives us single-binary distribution for `spekk next` and establishes the Go project structure that all subsequent modules build on.

## Strategy

- Go binary produces **identical JSON output** to the Node parser for the same input specs
- Node CLI delegates to Go binary when present, falls back to JS parser if not
- Once validated, Node parser code is removed
- Existing parser test scenarios define acceptance criteria for the Go implementation

## Success Criteria

- `go build ./cmd/spekk` compiles a working binary
- `spekk next` returns identical JSON whether using Go or Node parser
- All validation rules (frontmatter, status, timestamps, branches, depends-on, duplicates) match Node behavior
- Priority algorithm, branch-aware filtering, and lock staleness produce same results
- Node parser code is deleted after Go parser is validated
