---
id: skill-command
parent: agent-install
created: 2026-06-11T00:00:00Z
priority: 1
status: done
branch: feat/agent-install
---

# `spekk skill` Exposes Layered Skill Discovery on the CLI

`spekk skill list <agent>` and `spekk skill show <agent> <name>` expose the existing `SkillResolver` so agents running in any harness can discover and fetch skill content without filesystem access to the spekk repo.

## Success Criteria

- `spekk skill list coach` prints one line per available skill: the skill name and its source (the resolving directory, or `embedded`), using `SkillResolver.ListSkills`.
- The listing excludes the spec doc that shares its directory's name (e.g. `coach-skills-system.md` inside `specs/coach-skills-system/`) — only actual skills are listed.
- `spekk skill show coach <name>` prints the skill's markdown content to stdout, using `SkillResolver.ResolveSkill` (which honors the layer order: `.spekk/skills/<agent>/` → `~/.spekk/skills/<agent>/` → install dir → embedded FS).
- Unknown skill name prints an error to stderr and exits 1.
- Missing arguments print usage to stderr and exit 1.
- `spekk skill --help` shows usage for both subcommands.
- `spekk help` lists the `skill` command.

**Tests:** `internal/cli/skill_test.go` (resolution already covered); command wiring exercised via `cmd/spekk` dispatch.
