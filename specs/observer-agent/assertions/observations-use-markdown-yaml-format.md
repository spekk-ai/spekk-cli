---
id: observations-use-markdown-yaml-format
parent: observer-agent
created: 2026-01-22T17:00:00Z
priority: 2
status: done
---

# Observations Use Markdown + YAML Frontmatter Format

Observation files follow the same format conventions as specs and assertions for consistency.

**Tests:** src/observer/__tests__/observation-format.test.js

## Success Criteria

- [x] Observations use YAML frontmatter with required fields
- [x] Frontmatter includes: id, created (ISO 8601), type (observation type)
- [x] Frontmatter includes: severity (low/medium/high), affected_specs (array)
- [x] Frontmatter includes: affected_files (array of file paths)
- [x] Body uses markdown format for human-readable findings
- [x] Format is consistent with existing spec parser expectations
- [x] Files are parseable by existing YAML/markdown tooling