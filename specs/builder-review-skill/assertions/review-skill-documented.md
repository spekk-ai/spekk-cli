---
id: review-skill-documented
parent: builder-review-skill
created: 2026-09-03T19:44:00Z
priority: 2
status: not_started
branch: feature/builder-review-skill
depends-on: review-skill-discoverable
---

# The Builder Prompt And The Docs Name The Review Skill

## Description

The coach prompt has an "Available Skills" section that names its built-ins and tells the session how to load one. The builder prompt has none, so a builder session does not know a skill exists. The docs page `docs/coach-skills.md` says the builder supports skills and lists no builder skill. The CLI reference lists the builder's flags and no skill.

## Success Criteria

- `specs/builder-agent/builder.prompt.md` has an `## Available Skills` section. It names `review`, states when to use it (after the assertions for a feature are `done`, before the push), and states how to load it (`spekk skill show builder review`). It is short: no copy of the lenses.
- `docs/coach-skills.md` has a `## Built-in builder skills` section with a `### Review` entry. The entry states the CLI form `spekk builder review`, the alias `review` → `review-skill`, the scope rule, and the six lenses as a list of one line each.
- `docs/cli-reference.md` shows `spekk builder review` in the `spekk builder` command block, with a one-line comment.
- The docs nav check that CI runs (`make` target or script that checks `zensical.toml` against `docs/`) still passes. No new page is added.
- Prose added by this change is one line per paragraph.
