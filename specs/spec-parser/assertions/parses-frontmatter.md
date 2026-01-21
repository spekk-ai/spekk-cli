---
id: parses-frontmatter
parent: spec-parser
created: 2026-01-20T16:25:00Z
priority: 1
status: done
---

# Parser Must Parse YAML Frontmatter

## What Must Be True

The parser must correctly parse YAML frontmatter from spec and assertion files.

## Requirements

### YAML Structure
- Files MUST start with YAML frontmatter between `---` delimiters
- YAML must be well-formed and parseable
- Content after frontmatter is markdown

### Expected Format
```markdown
---
id: example-spec
created: 2026-01-20T15:40:00Z
priority: 1
---

# Spec Title

Markdown content here...
```

### Error Handling
Parser should handle and report errors for:
- Missing frontmatter delimiters
- Malformed YAML syntax
- Invalid YAML structure
- Files that don't start with frontmatter

### Success Criteria

- ✅ Parser extracts YAML frontmatter correctly
- ✅ Parser separates frontmatter from markdown content  
- ✅ Parser handles well-formed YAML
- ✅ Parser reports clear errors for malformed YAML
- ✅ Parser works with both spec and assertion files

**Tests:** `src/parser/__tests__/spec-parser.test.js`