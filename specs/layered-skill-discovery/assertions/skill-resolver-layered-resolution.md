---
id: skill-resolver-layered-resolution
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
status: done
---

# SkillResolver Resolves Skills in Layered Order

## Description

The `SkillResolver` class discovers skills using a layered resolution order: local (`.spekk/skills/{agent}/`) > global (`~/.config/spekk/skills/{agent}/`) > package (`specs/coach-skills-system/` or `specs/builder-skills/`). First match wins.

## Success Criteria

- `SkillResolver` is a class in `src/cli/skill-resolver.js`
- Constructor accepts `{ homeDir, cwd }` for testability, defaulting to `os.homedir()` and `process.cwd()`
- `resolveSkill(agentName, subcommand)` returns `{ name, content, source }` or `null`
- Resolution checks local dir first, then global, then package
- A local skill with the same filename as a global skill takes precedence
- A local or global skill with the same filename as a package skill takes precedence
- Returns `null` when no skill matches in any layer
- Package skill directories are: `specs/coach-skills-system/` for coach, `specs/builder-skills/` for builder
