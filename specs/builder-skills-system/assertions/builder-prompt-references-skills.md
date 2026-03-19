---
id: builder-prompt-references-skills
parent: builder-skills-system
created: 2026-03-19T18:04:00Z
priority: 1
status: done
depends-on: prompt-resolver-supports-per-agent-skills
branch: feature/code-quality-qa
---

# Builder prompt includes a skills section

The builder agent prompt (`specs/builder-agent/builder.prompt.md`) includes a section explaining that contextual skills are available and how to use them, mirroring how the coach prompt references coach skills.

## Success Criteria

- Builder prompt has an "Available Skills" or "Skills" section
- Section explains: skills are markdown files that provide domain-specific implementation patterns
- Section explains: when working on an assertion, check if the assertion content matches any skill triggers
- Section explains: if a skill matches, read the skill file and follow its workflow/patterns during implementation
- Section lists the skills directory path and explains the layered resolution (package → global → local)
- Section explains: project-local skills in `.spekk/builder-skills/` extend or override package skills
