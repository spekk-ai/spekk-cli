---
id: go-parser-validates-specs
parent: golang-parser
created: 2026-04-05T12:02:00Z
priority: 1
status: in_progress
locked-by: builder-Paris-MacBook-Pro-2-local-30697-1775423276
depends-on: go-parser-reads-specs
branch: feature/golang-parser
---

# Go parser validates spec files

The Go parser enforces the same validation rules as the Node parser, producing identical error messages for invalid input.

## Success Criteria

- Required fields enforced: specs need `id`, `created`, `priority`; assertions also need `parent`
- ID format validated as kebab-case (`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
- Priority validated as 1, 2, or 3 only
- Status validated as one of: `not_started`, `in_progress`, `done`, `draft`, `failed`
- Timestamps validated as ISO 8601 format (`YYYY-MM-DDTHH:MM:SSZ`)
- Branch field validated: no leading/trailing slashes, valid git branch characters, warns on non-standard patterns
- Duplicate spec IDs across groups detected and reported
- Duplicate assertion IDs within a spec detected and reported
- Assertion `parent` field references an existing spec
- `depends-on` field validated: must be kebab-case string, must reference existing assertion, no self-references
- Circular dependencies detected and reported with cycle path
- Folder structure enforced: no flat `.md` files with frontmatter at `specs/` root, spec directories must contain `{dir-name}.md`
