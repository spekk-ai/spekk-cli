---
id: run-observer-detects-skill
parent: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 1
status: done
depends-on: skill-resolver-includes-observer
branch: feature/observer-skill-discovery
---

# RunObserver Detects and Activates Skills

## Description

When a user runs `spekk observer <skill-name> [args...]` and `<skill-name>` resolves to an observer skill, the observer launches with the skill content inlined into the activation message. When no skill is provided (or the first arg isn't a recognizable skill), the observer runs in its default monitoring mode.

## Success Criteria

- `RunObserver` in `internal/agent/observer.go` checks the first positional arg against `SkillResolver.ResolveSkill("observer", arg)` before parsing flags as monitoring options
- If a skill resolves, `BuildSkillMessage(installDir, "observer", skillName, args)` is called and the result is appended to the activation message
- The activation message contains the skill content wrapped in `<skill-content>` tags (delegated through the existing `BuildSkillMessage` helper)
- If no skill is provided or the first arg starts with `-` (flag) or doesn't match any skill, observer falls back to its existing flag-parsing behavior (`--interval`, `--quiet`)
- Skill invocation does not break existing observer flag behavior — `spekk observer --interval 60` still works
- Tests exist in `internal/agent/` verifying both the skill-found and skill-not-found paths for observer

**Tests:** internal/agent/observer_test.go
