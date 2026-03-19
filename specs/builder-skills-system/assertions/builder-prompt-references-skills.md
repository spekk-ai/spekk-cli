---
id: builder-prompt-references-skills
parent: builder-skills-system
created: 2026-03-19T18:04:00Z
priority: 1
status: not_started
depends-on: builder-skills-directory-exists
branch: feature/code-quality-qa
---

# Builder prompt references the skills directory

The builder agent prompt (`specs/builder-agent/builder.prompt.md`) includes a section explaining that contextual skills are available and how to use them.

## Success Criteria

- Builder prompt includes a "Skills" or "Available Skills" section
- Section explains: skills are markdown files in `specs/builder-skills-system/`
- Section explains: when working on an assertion, check if any skill triggers match the assertion content
- Section explains: if a skill matches, read the skill file and follow its workflow/patterns during implementation
- Builder prompt references the skills directory path
- The `PromptResolver` `createActivationMessage` includes the builder skills directory path (similar to how it includes `skillsDir` for coach)
