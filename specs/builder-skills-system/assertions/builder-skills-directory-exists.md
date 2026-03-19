---
id: builder-skills-directory-exists
parent: builder-skills-system
created: 2026-03-19T18:03:00Z
priority: 1
status: not_started
branch: feature/code-quality-qa
---

# Builder skills directory exists with skill format documentation

The directory `specs/builder-skills-system/` exists and contains a parent spec that documents the skill file format and how the builder loads skills.

## Success Criteria

- Directory `specs/builder-skills-system/` exists (already created as part of this spec)
- Parent spec `builder-skills-system.md` documents:
  - Skill file format (triggers, workflow, patterns sections)
  - How the builder discovers and loads skills
  - That skills are markdown — no code, no loaders, no registries
- Pattern is consistent with coach skills system (`specs/coach-skills-system/coach-skills-system.md`)
