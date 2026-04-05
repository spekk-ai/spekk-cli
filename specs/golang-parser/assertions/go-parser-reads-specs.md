---
id: go-parser-reads-specs
parent: golang-parser
created: 2026-04-05T12:01:00Z
priority: 1
status: not_started
depends-on: go-project-structure
branch: feature/golang-migration
---

# Go parser reads and parses spec directories

The Go parser walks `specs/`, reads YAML frontmatter from markdown files, and builds the spec + assertion tree.

## Success Criteria

- Parser discovers all spec group directories under `specs/`
- Parser reads `{spec-id}/{spec-id}.md` as parent spec files
- Parser reads `{spec-id}/assertions/*.md` as assertion files
- YAML frontmatter extracted correctly: id, parent, created, priority, status, depends-on, branch, locked-by
- Markdown content (after frontmatter) preserved
- First H1 heading extracted as title
- CRLF line endings normalized to LF before parsing
- Files without `---` frontmatter delimiter are silently skipped
- Malformed files logged as warnings to stderr and skipped (not fatal)
- Specs without an `assertions/` directory logged as warning and skipped
- Running against this project's own `specs/` produces correct spec tree
