---
id: go-parser-json-matches-node
parent: golang-parser
created: 2026-04-05T12:04:00Z
priority: 1
status: not_started
depends-on: go-parser-computes-state
branch: feature/golang-migration
---

# Go parser JSON output matches Node parser

The Go parser produces semantically identical JSON output to the Node parser for all output modes.

## Success Criteria

**Next assertion output** (`spekk next`):
- JSON contains: `type`, `id`, `parent`, `file`, `priority`, `status`, `branch`, `created`, `dependsOn`, `lockedBy`, `title`, `content`, `spec` (with `id`, `file`, `title`)
- Field names use camelCase (matching Node convention): `dependsOn`, `lockedBy`

**Hierarchy output** (`spekk next --all`):
- JSON contains: `type: "hierarchy"`, `specs` array, `observations` array
- Each spec has: `id`, `title`, `status`, `priority`, `file`, `assertions` array
- Assertions sorted by priority then id; specs sorted by priority then id

**Complete output** (no remaining work):
- `{ "type": "complete", "status": "complete", "message": "All specifications are complete" }`

**Empty output** (no specs found):
- `{ "status": "empty", "message": "No specifications found in specs/ directory" }`

**Error output** (validation failures):
- `{ "error": true, "message": "..." }` with non-zero exit code

**Validation test:**
- Running both Node and Go parsers against this project's `specs/` produces identical JSON (ignoring whitespace)
