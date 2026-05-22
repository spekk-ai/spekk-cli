---
id: install-global-scope-writes-to-home
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: not_started
depends-on: install-fetches-from-official-registry
branch: feature/skill-install-system
---

# Install with --global Writes to Home Directory

## Description

`spekk install <agent> <skill> --global` writes to `~/.spekk/skills/<agent>/<skill>.md` so the skill is available across every project for the current user. This matches the global layer the `SkillResolver` already reads from.

## Success Criteria

- `spekk install builder my-skill --global` writes to `~/.spekk/skills/builder/my-skill.md` (resolved against `os.UserHomeDir()`)
- The home-side `.spekk/skills/<agent>/` tree is created with mode 0755 if missing
- A globally-installed skill is discoverable via `SkillResolver.ResolveSkill(agent, "my-skill")` without code changes to the resolver
- The command prints the absolute path under the user's home dir on success
