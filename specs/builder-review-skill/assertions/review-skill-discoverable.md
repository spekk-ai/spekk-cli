---
id: review-skill-discoverable
parent: builder-review-skill
created: 2026-09-03T19:42:00Z
priority: 1
status: done
branch: feature/builder-review-skill
depends-on: review-skill-embedded
---

# The Review Skill Resolves And Is Listed As `review`

## Description

The resolver in `internal/cli/skill.go` finds a skill by filename stem, by alias, or by frontmatter `id`. The filename stem is `review-skill` and the invocation name is `review`, so the builder alias map needs the entry `review` → `review-skill`, exactly as the observer map has `prune` → `prune-skill`. With it, help shows the clean name and both names resolve.

The builder help falls back to a shared example block that shows `spekk builder meeting`, which is a coach skill. With a real builder skill, the builder gets its own example block.

**Tests:** internal/cli/skill_test.go (mirror `TestResolveSkill_ObserverEmbeddedPruneAlias` for `builder` and `review`), internal/agent/launcher_test.go (the builder help lists `review` and its example block names `spekk builder review`).

## Success Criteria

- `legacyAliases["builder"]` in `internal/cli/skill.go` contains `"review": "review-skill"`.
- `SkillResolver.ResolveSkill("builder", "review")` and `ResolveSkill("builder", "review-skill")` both return the skill, from the embedded FS in a clean checkout and from `specs/builder-skills/` in a source tree.
- `SkillResolver.ListSkills("builder")` includes `review-skill`.
- `spekk builder --help` lists `review` under `AVAILABLE SKILLS:`, not the raw stem, and its `EXAMPLES:` block names `spekk builder review`. It no longer shows the `meeting` examples.
- `spekk skill show builder review` prints the skill content.
- `spekk builder review` launches the builder with the skill inlined, in the same way `spekk observer prune` does. No change to `RunBuilder` is needed for this; the assertion is that it works.
- Existing coach and observer skills and aliases are unchanged.
