---
id: skill-keeps-disciplines
parent: dev-loop-single-session
created: 2026-07-27T00:00:00Z
priority: 1
status: done
depends-on: skill-single-session-flow
---

# The Skill Keeps the Standing Disciplines

## Description

The change is the execution model, not the standards. The rewritten skill keeps
the disciplines that the old skill had. A reader must still get the same quality
bar.

## Success Criteria

- The skill keeps the lean-and-simple mandate: build the simplest thing that
  solves the real problem, not general or configurable machinery.
- The skill keeps the ground-in-current-source rule: read the real code, not a
  summary.
- The skill keeps the verify-for-real rule: check the assertion's real success
  criteria, including a live check when the assertion calls for one; do not stop
  at "it compiles".
- The skill keeps the test bar: tests must be lean and high value; delete
  low-value or redundant tests.
- The skill keeps the assertion-status rule: mark an assertion `not_started` when
  it is buildable, and `draft` only for a real, named design gap.
