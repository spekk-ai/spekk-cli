---
id: prune-skill-markdown-exists
parent: observer-prune-skill
created: 2026-07-25T12:00:00Z
priority: 1
status: done
branch: feature/observer-prune-skill
---

# Prune Skill Markdown Exists With Conservative Deletion Lenses

## Description

`specs/observer-skills/prune-skill.md` exists as a package observer skill that
surfaces deletion / simplification candidates for human review. It follows the
`coverage-gap-skill.md` template exactly (same frontmatter shape, same body
section conventions) and defines its detection lenses **conservatively** —
deletion advice is dangerous, so the skill must optimize for high precision
(evidence-backed, no false positives) over recall.

This assertion covers the skill markdown's content only. Embedding
(`embedded.go`), the allowed-type registration in the two contract docs, and
discovery are separate assertions.

**Tests:** embedded_test.go (a convention check that the shipped
`prune-skill.md` has the required frontmatter fields and the required body
sections — mirror the existing `TestEmbeddedFS_ObserverCoverageGapSkill` string
assertions in `embedded_test.go`; do not invent new lint machinery).

## Success Criteria

- File `specs/observer-skills/prune-skill.md` exists.
- Frontmatter has the four fields `coverage-gap-skill.md` uses, and no others:
  - `id: prune` (kebab-case; this is the resolver key)
  - `description:` (one line describing the removal/consolidation lens)
  - `created:` (ISO-8601, UTC)
  - `priority:` (integer)
- Body contains, in this order, the section headings the coverage-gap template
  uses: `## Triggers`, `## Workflow`, `## Output Format`, `## Validation`,
  `## Examples`.
- The `## Workflow` defines exactly these four detection lenses, each described
  as evidence-backed and conservative:
  - **(a) Deletion candidate** — code with **no owning assertion AND no caller
    AND no test**. This is the removal counterpart to `coverage-gap`'s
    "document it" recommendation.
  - **(b) Duplication** — near-identical code that should be consolidated to a
    single source of truth.
  - **(c) Over-abstraction** — speculative generality (interfaces, options,
    layers) that has no current use.
  - **(d) Dead configuration** — flags, options, or config keys that are never
    read or exercised.
- The `## Workflow` explicitly states the skill is **recommend-only** and MUST
  NOT delete, edit, or move any code or spec — it only writes an observation
  (the observer's read-only contract, worded as in `coverage-gap-skill.md`).
- The `## Workflow` **cross-references `coverage-gap`**: the deletion lens (a)
  keys off the same orphan signal as `coverage-gap` but makes the **opposite**
  recommendation (remove vs document); the skill states that the human decides
  document-vs-remove, and that `prune` should not re-report a region already
  captured by a recent `coverage-gap` observation.
- The `## Workflow` states a **bias toward NOT flagging**: when any plausible
  reason to keep the code exists (public/exported API surface, a caller the
  scan may not have resolved, reflection/plugin/generated usage, or a recent
  `coverage-gap` finding), the region is omitted, not flagged.
- The `## Output Format` follows the Observation Output Contract: writes to
  `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`; frontmatter includes
  `skill: prune`, `type: prune_candidate`, `severity: low|medium|high`,
  `affected_specs`, `affected_files`, `id`, `created`; body sections are
  `## Issue Description`, `## Evidence`, `## Impact`, `## Recommendation` in
  that order. Because the file lands under `observations/prune/`, `consolidate`
  and `DIGEST.md` pick it up automatically — no consolidate code change.
- The `## Validation` section asserts: output path is
  `observations/prune/...`; frontmatter uses `type: prune_candidate`; the skill
  wrote no code/spec files; Evidence lists concrete file paths (and caller /
  test / assertion absence) — no vague claims.
- The skill mirrors `coverage-gap`'s exclusions: `*_test.go`, generated code,
  and thin `cmd/` entry points are out of scope.
