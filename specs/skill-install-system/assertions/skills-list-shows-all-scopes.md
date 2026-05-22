---
id: skills-list-shows-all-scopes
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 2
status: not_started
---

# `spekk skills list` Shows Skills Across All Scopes

## Description

`spekk skills list <agent>` displays every skill currently available to the named agent — local + global + embedded — using the existing `SkillResolver.ListSkills(agent)` method. Each entry shows the skill name and its source directory so users can tell which scope is shadowing what.

## Success Criteria

- `spekk skills list coach` calls `SkillResolver.ListSkills("coach")` and prints the result
- Output includes skills from local (`.spekk/skills/<agent>/`), global (`~/.spekk/skills/<agent>/`), and embedded scopes
- Local shadows global shadows embedded (matches `ListSkills` dedup behavior — this assertion is about display, not changing resolver semantics)
- Each row shows the skill name and its source (directory path, or `(embedded)`)
- Passing an unknown agent is rejected with the same validation as install
- When no skills are available for the agent, prints a "no skills found" message and exits 0
- `spekk skills` (no subcommand) prints usage listing `list`
