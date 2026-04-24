---
id: update-installation-docs
parent: golang-cleanup
created: 2026-04-05T12:34:00Z
priority: 3
status: done
depends-on: update-ci-for-go
branch: feature/golang-migration
---

# Installation and documentation updated for Go binary

All documentation reflects the Go binary distribution model instead of npm package.

## Success Criteria

- README installation instructions show `go install` or binary download
- No references to `npm install`, `node`, or `package.json` in docs
- CLAUDE.md updated: `npm run next` → `spekk next`, etc.
- Spec prompts (coach, builder, observer) updated to reference Go binary
- Contributing guide describes Go development workflow (`go build`, `go test`)
- Gemfury publish spec marked as superseded (Go uses GitHub releases, not npm registry)
