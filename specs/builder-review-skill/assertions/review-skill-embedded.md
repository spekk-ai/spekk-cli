---
id: review-skill-embedded
parent: builder-review-skill
created: 2026-09-03T19:41:00Z
priority: 1
status: done
branch: feature/builder-review-skill
depends-on: review-skill-markdown-exists
---

# The Review Skill Is Embedded In The Binary

## Description

The `//go:embed` directive in `embedded.go` is an explicit-path embed. A new skill file is invisible to the installed binary until its path is added. Without this, `spekk builder review` resolves only from a source checkout.

**Tests:** embedded_test.go (mirror `TestEmbeddedFS_ObserverPruneSkill`: read `specs/builder-skills/review-skill.md` from `EmbeddedFS`, and assert the frontmatter fields and the five section headings that `review-skill-markdown-exists` requires).

## Success Criteria

- The `//go:embed` directive in `embedded.go` includes `specs/builder-skills/review-skill.md`. No existing path is removed.
- `fs.ReadFile(spekk.EmbeddedFS, "specs/builder-skills/review-skill.md")` succeeds and returns the exact bytes of the source file.
- The test asserts `id: review` and the five headings in order.
