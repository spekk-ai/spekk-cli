---
id: parser-handles-crlf-line-endings
parent: spec-parser
created: 2026-03-16T00:00:00Z
priority: 1
status: done
---

# Parser handles CRLF line endings

Spec files with Windows-style CRLF line endings are parsed correctly. The frontmatter delimiter detection does not break when `\r\n` is present.

## Success Criteria

- Frontmatter parsing normalizes line endings before processing
- A `.gitattributes` file enforces LF for `specs/**/*.md`
