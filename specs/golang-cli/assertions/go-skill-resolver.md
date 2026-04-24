---
id: go-skill-resolver
parent: golang-cli
created: 2026-04-05T12:13:00Z
priority: 1
status: done
depends-on: go-command-router
branch: feature/golang-migration
---

# Go skill resolver implements layered skill discovery

The Go skill resolver discovers and loads skill files using the same layered resolution as the Node implementation.

## Success Criteria

**Resolution order (first match wins):**
1. Local: `.spekk/skills/{agent}/*.md`
2. Global: `~/.spekk/skills/{agent}/*.md`
3. Package: `specs/{agent}-skills-system/*.md` or `specs/builder-skills/*.md`

**Skill matching:**
- Direct filename match: `{name}.md`
- Legacy alias match: e.g., `meeting` → `meeting-notes-to-specs-skill`
- Frontmatter `id` field match (scans all `.md` files in directory)

**Listing:**
- `listSkills(agent)` returns all skills with deduplication (local shadows global shadows package)
- `listAliases(agent)` returns legacy alias map

**Legacy aliases preserved:**
- Coach: `meeting` → `meeting-notes-to-specs-skill`, `coordinate` → `coordinator-skill`
