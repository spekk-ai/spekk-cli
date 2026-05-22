---
id: install-default-scope-is-local
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: in_progress
depends-on: install-fetches-from-official-registry
branch: feature/skill-install-system
locked-by: builder-Paris-MacBook-Pro.local-53774-1779478819
---

# Install Defaults to Local Scope

## Description

When neither `--global` nor `--local` is passed, `spekk install` writes the skill to `<cwd>/.spekk/skills/<agent>/<skill>.md`. This matches the precedence the existing `SkillResolver` already gives to local files and is what users expect from `npm install`-style ergonomics.

## Success Criteria

- `spekk install coach meeting-notes` (with no scope flags) writes the file to `<cwd>/.spekk/skills/coach/meeting-notes.md`
- The `.spekk/skills/<agent>/` directory tree is created with mode 0755 if it doesn't exist
- The file is written with mode 0644
- The command prints a one-line confirmation that includes the absolute path written
- The command exits 0 on success
