---
id: review-skill-markdown-exists
parent: builder-review-skill
created: 2026-09-03T19:40:00Z
priority: 1
status: done
branch: feature/builder-review-skill
---

# The Review Skill Markdown Exists With Six Lenses

## Description

`specs/builder-skills/review-skill.md` exists as a package builder skill. It follows the observer skill template (`specs/observer-skills/prune-skill.md`): the same four frontmatter fields, the same five body sections in the same order. Its body is the review procedure: a scope rule, six lenses with a remedy each, the report, and the push gate.

This assertion covers the file's content only. Embedding, discovery, the dev-loop change, and the docs are separate assertions.

**Tests:** embedded_test.go (a convention check that the shipped file has the frontmatter fields and the section headings, mirroring `TestEmbeddedFS_ObserverPruneSkill`; that test lands with `review-skill-embedded`).

## Success Criteria

- File `specs/builder-skills/review-skill.md` exists.
- Frontmatter has exactly these four fields: `id: review`, `description:` (one line), `created:` (ISO-8601, UTC), `priority:` (integer).
- The body contains, in this order, the headings `## Triggers`, `## Workflow`, `## Output Format`, `## Validation`, `## Examples`.
- `## Workflow` states the scope rule: the assertions marked `done` on the current branch since it left its base branch, plus the diff from that base to `HEAD`. It names the git commands that find both. It states that on the base branch itself the user names the commit range.
- `## Workflow` defines exactly six lenses, in this order, each with its remedy:
  - (1) Every success criterion of every in-scope assertion is checked against the real code. A criterion with non-obvious behavior is traced with one concrete input. An unmet criterion is fixed in the session, or the assertion is set to `failed` when it cannot be.
  - (2) Every test earns its place. A test that passes when the behavior it pins is broken, that restates the implementation, that duplicates another test, or that exercises a mock instead of the real path is deleted.
  - (3) Nothing beyond what the assertions ask for. Generality, configuration, and abstraction no assertion requires are removed. A diff hunk no assertion accounts for is reverted, or the reason it stays is stated.
  - (4) Errors are loud. A dropped, defaulted, or broadly caught error is a defect and is fixed.
  - (5) The spec tree is sound: `spekk validate` exits 0, `spekk next` succeeds, no stale `locked-by`, and each `**Tests:**` link points to an existing file.
  - (6) The diff is fit to publish: no secret, no private name, no reference to another repository.
- `## Workflow` states that the review fixes what it finds, because the builder role may write code, and that it writes no observation file.
- `## Workflow` states the push gate: the push waits for the review.
- `## Output Format` describes the session report: each in-scope assertion with a verdict, what was fixed, what was deleted, what stays open. It states that no file under `observations/` is written.
- `## Validation` lists the checks a reviewer runs before it reports: `spekk validate` and `spekk next` succeed, the test suite passes, and every assertion in scope is `done` or `failed`.
- Prose in the file is one line per paragraph. Lists and code fences keep their own lines.
