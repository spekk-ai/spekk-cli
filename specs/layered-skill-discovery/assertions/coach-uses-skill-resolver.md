---
id: coach-uses-skill-resolver
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
status: done
depends_on:
  - skill-resolver-layered-resolution
  - legacy-aliases-backward-compat
---

# Coach CLI Uses SkillResolver for Skill Detection

## Description

The coach CLI (`src/coach/cli.js`) delegates all skill resolution to `SkillResolver` instead of using a hardcoded `SKILL_MAP`. The `buildSkillActivationMessage()` function accepts an optional `SkillResolver` instance for testability. `showHelp()` dynamically lists available coach skills.

## Success Criteria

- `src/coach/cli.js` imports and uses `SkillResolver` from `src/cli/skill-resolver.js`
- `buildSkillActivationMessage()` delegates to `resolver.resolveSkill('coach', subcommand)`
- `launchCoachAgent()` uses `skillResolver.resolveSkill('coach', subcommand)` to detect skills
- `showHelp()` calls `skillResolver.listSkills('coach')` and dynamically lists available skills
- Skill content is inlined within `<skill-content>` tags in the activation message
- Coach-specific behavior (transcript handling for `meeting`) remains in coach CLI
- Adding a new coach skill requires only creating a markdown file — no JS changes needed
