---
id: parity-with-coach-and-builder
parent: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 3
status: not_started
depends-on: run-observer-detects-skill
---

# Observer Has Skill Parity With Coach and Builder

## Description

The observer agent is extensible and overrideable in the same ways coach and builder are. From a user's perspective, working with observer skills feels identical to working with coach or builder skills — same directory layout, same resolution order, same CLI invocation pattern, same help output structure.

## Success Criteria

- A new observer skill added to `.spekk/skills/observer/my-skill.md` is invocable as `spekk observer my-skill` without any code changes
- A global observer skill at `~/.spekk/skills/observer/my-skill.md` works across all projects
- A local skill shadows a global skill of the same name (verified via `ListSkills` returning the local source)
- Layered prompt customization (already working) and layered skill discovery (this spec) both function for observer — `.spekk/observer.prompt.md` and `.spekk/skills/observer/*.md` can coexist
- The pattern is documented somewhere a user would find it (README, `specs/layered-skill-discovery/`, or observer agent spec)
- Adding a future agent type requires only registering it in `validAgents` (prompts), `packageSkillDirNames` (skills), and `legacyAliases` — no other code paths special-case agent names
