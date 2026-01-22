---
id: spec-parser-handles-observations
parent: observer-agent
created: 2026-01-22T17:00:00Z
priority: 3
status: not_started
---

# Spec Parser Handles Observation Files

The existing spec parser is extended to parse and validate observation files alongside specs and assertions.

## Success Criteria

- [ ] Parser scans `observations/` directory for `.md` files
- [ ] Parser validates observation YAML frontmatter structure
- [ ] Parser includes observations in JSON output with separate section
- [ ] Parser validates required observation fields (id, created, type, severity)
- [ ] Parser validates affected_specs references point to existing specs
- [ ] Parser validates affected_files references point to existing files
- [ ] Observation parsing errors are reported clearly