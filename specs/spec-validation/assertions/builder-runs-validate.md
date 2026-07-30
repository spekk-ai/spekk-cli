---
id: builder-runs-validate
parent: spec-validation
created: 2026-07-23T21:10:51Z
priority: 2
status: done
depends-on: validate-command
---

# Builder prompt requires a clean `spekk validate` before committing

`specs/builder-agent/builder.prompt.md` instructs the builder to run
`spekk validate` before it commits, and to treat a non-zero exit as a blocker
that must be fixed before the commit is made.

## Success criteria

- The builder prompt, in the pre-commit region (the "Validate System Health" /
  "Commit and Push" area, steps 6–7), instructs the builder to run
  `spekk validate` as part of finishing an assertion.
- It states the contract plainly: a **non-zero exit means the spec tree is
  invalid** (e.g. a lock left dangling, a status/lock mismatch, a malformed
  frontmatter block the builder just wrote) and the builder must resolve every
  reported failure before committing — a failing `validate` is never committed.
- This complements, not replaces, the existing `spekk next` system-health check.
- Additive prompt edit only; no other builder behavior changes.

**Note:** Depends on `validate-command` — the instruction is meaningless until
the command exists. This is prompt prose; it does not wire `validate` into the
Go loop orchestration (that would be more than the gap requires).
