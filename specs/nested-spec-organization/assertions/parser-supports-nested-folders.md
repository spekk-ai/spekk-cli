---
id: parser-supports-nested-folders
parent: nested-spec-organization
created: 2026-01-22T17:15:00Z
priority: 2
status: draft
---

# Parser Supports Nested Folder Structures

The spec parser can detect and parse nested folder hierarchies with group-level specifications.

## Success Criteria

- [ ] Parser recursively scans subdirectories under `specs/`
- [ ] Parser identifies group specifications by `type: group` frontmatter
- [ ] Parser validates nested folder structure rules
- [ ] Parser computes group status from child spec statuses
- [ ] Parser maintains backward compatibility with flat structure
- [ ] Parser includes groups in JSON output with child relationships