---
id: observations-use-markdown-yaml-format
parent: observer-agent
created: 2026-01-22T17:00:00Z
priority: 2
status: not_started
---

# Observations Use Markdown + YAML Frontmatter Format

Observation files follow the same format conventions as specs and assertions for consistency.

## Success Criteria

- [ ] Observations use YAML frontmatter with required fields
- [ ] Frontmatter includes: id, created (ISO 8601), type (observation type)
- [ ] Frontmatter includes: severity (low/medium/high), affected_specs (array)
- [ ] Frontmatter includes: affected_files (array of file paths)
- [ ] Body uses markdown format for human-readable findings
- [ ] Format is consistent with existing spec parser expectations
- [ ] Files are parseable by existing YAML/markdown tooling