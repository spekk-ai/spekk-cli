---
id: parser-skips-malformed-assertions
parent: robust-error-handling
created: 2026-03-16T18:00:00Z
priority: 1
status: not_started
depends-on: parser-skips-specs-without-assertions-dir
---

# Parser Skips Malformed Assertion Files

**Closes:** #41

## What Must Be True

When individual assertion files have invalid or missing frontmatter, the parser warns to stderr and skips them — it never crashes the CLI or returns an error JSON response.

## Success Criteria

- Assertion files missing required fields (id, parent, status, priority) are skipped with a stderr warning
- Assertion files with unparseable YAML frontmatter are skipped with a stderr warning
- The parent spec's computed status is derived only from its successfully-parsed assertions
- `spekk next` returns the next valid assertion even when other assertion files are malformed
- A spec with zero parseable assertions (all malformed) is treated the same as a spec with no assertions
