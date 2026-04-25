---
id: skill-name-derivation
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
status: done
---

# Skills Are Identified by Filename Stem and Frontmatter ID

## Description

A skill's subcommand name is derived from its filename stem (minus `.md`). Additionally, the `id` field in YAML frontmatter serves as an alternative lookup name, so `api-audit-tool.md` with `id: api-audit` can be invoked as `spekk builder api-audit`.

## Success Criteria

- Filename stem is the primary skill name: `my-validator.md` → subcommand `my-validator`
- Frontmatter `id` field is checked as fallback when no filename match exists
- Only `.md` files are considered as skills (non-`.md` files are ignored)
- `listSkills()` returns skill names derived from filename stems
- `listSkills()` deduplicates across layers — earlier layers (local) take precedence
