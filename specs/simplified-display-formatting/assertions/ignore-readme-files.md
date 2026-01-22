---
id: ignore-readme-files
parent: simplified-display-formatting
created: 2026-01-22T22:45:00Z
priority: 1
status: done
---

# Ignore Markdown Files Without Frontmatter

The parser crashes on any .md file without YAML frontmatter. Files without frontmatter are NOT specs and should be silently ignored.

## What Must Be True

Parser silently skips any .md file that doesn't start with YAML frontmatter.

### Simple Rule

**If a .md file doesn't start with `---`, skip it entirely.**

- No error thrown for missing frontmatter
- File is completely ignored during parsing
- Only files with proper `---` YAML frontmatter are parsed as specs/assertions

### Examples to Ignore

- README.md, docs.md, notes.md (no frontmatter)
- Any .md file without `---` at the start
- Documentation files mixed in specs directories

### Error Prevention

- `spekk builder` should work even with README files present
- Parser should not require YAML frontmatter on documentation files
- Builder loops should not crash on mixed content in specs directories

## Success Criteria

- ✅ Parser skips any .md file not starting with `---`  
- ✅ `spekk builder` works in `/Users/william/thinknimble/spekk/` without crashes
- ✅ No error thrown for files without YAML frontmatter
- ✅ README.md and other docs are silently ignored

**Tests:** src/parser/__tests__/ignore-readme-files.test.js