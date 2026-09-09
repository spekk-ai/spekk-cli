---
id: dev-loop-verify-uses-review-skill
parent: builder-review-skill
created: 2026-09-03T19:43:00Z
priority: 1
status: done
branch: feature/builder-review-skill
depends-on: review-skill-markdown-exists
---

# The Dev Loop's Verify Phase Loads The Review Skill

## Description

Phase 3 of `specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md` is one paragraph, and it points at a skill that does not ship with spekk. After this change it loads the review skill and follows it in the session, and it names the fresh-context path for the case where independence matters more than context.

The criteria of the done assertion `skill-keeps-disciplines` stay true: the phase still states the test bar in the skill body.

**Tests:** embedded_test.go (extend `TestEmbeddedFS_SpekkDevLoopSkill` with the two strings below; no new test function).

## Success Criteria

- Phase 3 tells the reader to run `spekk skill show builder review` and to follow the output in the same session.
- Phase 3 states the fresh-context path: for a high-stakes or large change, run `spekk builder review` instead, because a self-review loses independence. It frames this as the exception, in the same register as the sub-agent escalation.
- Phase 3 keeps the test bar in one sentence: tests must be lean and high value, and low-value or redundant tests are deleted.
- The skill body no longer references `code-quality-principles`.
- Phase 3 keeps the push gate: push only after the review.
- The skill's frontmatter `name` and `description` are unchanged, so `skill-installs-unchanged` stays true.
- `TestEmbeddedFS_SpekkDevLoopSkill` asserts the strings `spekk skill show builder review` and `spekk builder review` are present.
- The paragraphs this change touches are one line each. Untouched paragraphs keep their current wrapping.
