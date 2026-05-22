---
id: skill-resolver-includes-observer
parent: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 1
status: done
branch: feature/observer-skill-discovery
---

# SkillResolver Includes Observer in Package Directory Map

## Description

`SkillResolver` recognizes `observer` as a valid agent and knows where to find package-shipped observer skills. Without this, the resolver returns `nil` for any observer skill lookup regardless of the layer.

## Success Criteria

- `packageSkillDirNames` in `internal/cli/skill.go` includes an entry mapping `"observer"` to `"specs/observer-skills"`
- `SkillResolver.skillDirs("observer")` returns the three expected layered directories (local `.spekk/skills/observer`, global `~/.spekk/skills/observer`, package `specs/observer-skills`)
- `SkillResolver.ResolveSkill("observer", <skill-name>)` returns a resolved skill when one exists at any layer
- `SkillResolver.ListSkills("observer")` returns observer skills with proper layer-based shadowing (local shadows global shadows package)
- `legacyAliases` map contains an `"observer"` key (may be empty) so `ListAliases("observer")` never returns nil
- Embedded FS fallback also works for observer (skills baked into the binary are discoverable)

**Tests:** internal/cli/skill_test.go
