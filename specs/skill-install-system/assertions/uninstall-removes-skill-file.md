---
id: uninstall-removes-skill-file
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 2
status: not_started
---

# Uninstall Removes the Skill File from the Chosen Scope

## Description

`spekk uninstall <agent> <skill>` deletes the skill file at the chosen scope. Scope flags (`--local`, `--global`) match install. Uninstall never touches embedded skills or files outside the two on-disk scope directories.

## Success Criteria

- `spekk uninstall coach meeting-notes` removes `<cwd>/.spekk/skills/coach/meeting-notes.md` (local is default)
- `spekk uninstall coach meeting-notes --global` removes `~/.spekk/skills/coach/meeting-notes.md`
- If the target file doesn't exist, the command exits non-zero with a "not installed: <path>" error
- Uninstall never deletes files outside `<scope>/.spekk/skills/<agent>/`
- Uninstall does not affect embedded skills (the binary's compiled-in skill files remain available to `SkillResolver`)
- Passing an unknown agent is rejected with the same validation as install
- Omitting `<skill>` exits non-zero with usage
- Both `--global` and `--local` together is an error (mutually exclusive)
