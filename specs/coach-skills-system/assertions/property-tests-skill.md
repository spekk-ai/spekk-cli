---
id: property-tests-skill
parent: coach-skills-system
created: 2026-09-03T14:00:00Z
priority: 2
status: done
---

# Property Tests Skill

Coach has a property-tests skill that decides whether a promise in the specs deserves a property-based test and then writes the test for the right layer.

## Success Criteria

- `spekk coach property-tests` resolves to `specs/coach-skills-system/property-tests-skill.md`, and the alias `property-tests` resolves to the same file
- The skill ships in the binary through the embedded filesystem, and `embedded_test.go` checks that it declares `id: property-tests` and carries the Triggers, Workflow, and Validation sections
- The workflow puts a value gate before any code: a property restates a `done` assertion, needs search that a fixed-input test cannot supply, guards a failure that would matter, has evidence behind it, costs less than it is worth, and keeps the portfolio balanced
- The workflow names the two anti-patterns to refuse: exhaustive enumeration of a trivial finite space, and a property that duplicates a fixture test
- The workflow covers both layers, a browser explorer and a backend property library, with a study lens table, a choice of form, a clean run, a reach proof, and triage that ends in one issue per surviving violation
- `docs/coach-skills.md` documents the skill beside the other built-in coach skills
- The skill names no private project, host, account, or person

**Tests:** `embedded_test.go`, `TestEmbeddedFS_CoachPropertyTestsSkill`
