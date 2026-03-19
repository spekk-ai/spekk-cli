---
id: builder-skills-directory-exists
parent: builder-skills-system
created: 2026-03-19T18:03:00Z
priority: 1
status: not_started
branch: feature/code-quality-qa
---

# Builder skills directory exists at package level

The directory `specs/builder-skills-system/` exists in the spekk package and is shipped with npm, parallel to the existing `specs/coach-skills-system/`.

## Success Criteria

- Directory `specs/builder-skills-system/` exists in the package
- Parent spec `builder-skills-system.md` documents the skill file format (triggers, workflow, patterns sections) — consistent with `specs/coach-skills-system/coach-skills-system.md`
- `package.json` `files` array includes `specs/builder-skills-system/` so skills ship with the package
- Skills follow the same markdown structure as coach skills (no code, no loaders, no registries)
