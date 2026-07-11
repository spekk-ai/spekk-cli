---
id: builder-supports-skills
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
status: done
depends-on: skill-resolver-layered-resolution
---

# Builder CLI Supports Skill Subcommands

## Description

The builder CLI (`src/builder/cli.js`) now accepts a skill subcommand as the first positional argument. If it resolves via `SkillResolver`, the skill content is inlined into the activation message and Claude is launched once. Skills compose with flags: `spekk builder my-skill --once` works.

## Success Criteria

- `src/builder/cli.js` imports and uses `SkillResolver` from `src/cli/skill-resolver.js`
- First positional (non-flag) argument is extracted and checked as a potential skill name
- If skill resolves, content is inlined within `<skill-content>` tags in the activation message
- Claude is launched once with the skill-enriched activation message, then exits
- If the positional arg does not resolve as a skill, normal builder behavior continues
- `showHelp()` dynamically lists available builder skills from all layers
- `spekk builder api-audit` discovers and inlines a global skill from `~/.config/spekk/skills/builder/`
- Flag parsing is unaffected — `extractSkillArg()` correctly skips flags and their values
