---
id: prune-skill-discoverable
parent: observer-prune-skill
created: 2026-07-25T12:00:00Z
priority: 1
status: done
depends-on: prune-skill-embedded
branch: feature/observer-prune-skill
---

# Prune Skill Resolves And Is Listed As `prune`

## Description

Once the file exists and is embedded, discovery is handled by the existing resolver (`internal/cli/skill.go`) and the shared help helper (`internal/agent/launcher.go`) — no new wiring beyond a legacy alias. Because the filename stem (`prune-skill`) differs from the invocation name (`prune`), the `observer` alias map must map `prune` → `prune-skill` so help displays the clean invocation name, exactly as `coverage-gap` → `coverage-gap-skill` does today.

**Tests:** internal/cli/skill_test.go and internal/agent/launcher_test.go — mirror the existing coverage-gap discovery/help tests (`skill-resolver-includes-observer`, `observer-help-lists-skills`).

## Success Criteria

- `legacyAliases["observer"]` in `internal/cli/skill.go` contains `"prune": "prune-skill"` (alongside the existing `coverage-gap` entry).
- `SkillResolver.ResolveSkill("observer", "prune")` returns the prune skill (resolved from the embedded FS in a clean checkout, and from `specs/observer-skills/` in a source tree).
- `SkillResolver.ListSkills("observer")` includes the prune skill, and `spekk observer --help` (via `agent.ShowHelp(installDir, "observer")`) lists it under its invocation name `prune` in the `AVAILABLE SKILLS:` section — the raw stem `prune-skill` is NOT shown.
- `spekk observer prune` activates the skill (its content is inlined into the observer activation message), consistent with how `spekk observer coverage-gap` behaves.
- Existing observer skills (`coverage-gap`, `consolidate`) and observer flag behavior (`--quiet`) are unaffected — `prune` is additive.
