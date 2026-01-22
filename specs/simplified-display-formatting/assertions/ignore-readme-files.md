---
id: ignore-readme-files
parent: simplified-display-formatting
created: 2026-01-22T22:45:00Z
priority: 1
status: not_started
---

# Parser Should Ignore README Files

The parser fails when it encounters README.md files in specs directories that don't have YAML frontmatter.

## What Must Be True

Parser ignores documentation files that aren't specs or assertions.

### Files to Ignore

- **README.md** files in any specs directory
- **readme.md** files (case insensitive)
- Files in subdirectories that aren't `/assertions/`
- Non-spec documentation files

### Parser Behavior

- Only parse `.md` files that are either:
  - Spec files: `specs/{spec-id}/{spec-id}.md`  
  - Assertion files: `specs/{spec-id}/assertions/{assertion-id}.md`
- Skip all other `.md` files (README, docs, mockups, etc.)
- Don't error on files without YAML frontmatter if they're in ignored categories

### Error Prevention

- `spekk builder` should work even with README files present
- Parser should not require YAML frontmatter on documentation files
- Builder loops should not crash on mixed content in specs directories

## Success Criteria

- ✅ Parser ignores README.md files in specs directories
- ✅ `spekk builder` works in directories with README files  
- ✅ Only actual spec/assertion .md files are parsed for frontmatter
- ✅ No frontmatter errors for documentation files