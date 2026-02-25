---
id: lean-test-philosophy
parent: builder-agent
created: 2026-02-25T00:30:00Z
priority: 1
status: not_started
---

# Builder Follows Lean Testing Philosophy

The builder agent writes high-value tests only and actively removes redundant or low-value tests.

## Success Criteria

- Builder prompt includes explicit lean testing guidance
- Tests validate meaningful behavior, not implementation details
- No redundant tests — if two tests fail for the same reason, only one exists
- Trivial code (getters, pass-throughs, framework behavior) is not tested
- When the builder encounters redundant or low-value tests while working on an assertion, it deletes them
- Integration tests are preferred over unit tests when they cover the same workflow more efficiently
- Test suite runs fast and every test earns its place
