---
id: thin-role-skills
parent: roles-as-skills
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/roles-as-skills
depends-on: roles-split-agents-and-skills
---

# The Coach and Builder Skills Are Thin

## Description

The coach and builder skills do not embed the full role instructions. Each one
tells the session to run `spekk prompt <role>` and to follow the output. This
keeps the per-role overrides and lets a user run one role on its own.

## Success Criteria

- The coach skill body and the builder skill body each instruct the session to
  run `spekk prompt coach` or `spekk prompt builder`, and to adopt the output as
  the operating instructions for the session.
- Each skill has frontmatter with a `name` field (`spekk-coach` or
  `spekk-builder`) and a `description` field that scopes host delegation to a
  project with a `specs/` directory.
- For a command host (codex, cursor, copilot), the frontmatter is stripped, the
  same as the dev-loop skill.
- The skill body reuses the existing shim body text, so the "run spekk prompt,
  stand down without a specs/ directory" behavior does not change.
