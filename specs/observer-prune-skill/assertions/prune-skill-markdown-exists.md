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

`specs/observer-skills/prune-skill.md` exists as a package observer skill that surfaces deletion / simplification candidates for human review. It follows the `coverage-gap-skill.md` template exactly (same frontmatter shape, same body section conventions) and defines its detection lenses **conservatively** — deletion advice is dangerous, so the skill must optimize for high precision (evidence-backed, no false positives) over recall.

This assertion covers the skill markdown's content only. Embedding (`embedded.go`), the allowed-type registration in the two contract docs, and discovery are separate assertions.

**Tests:** embedded_test.go (a convention check that the shipped `prune-skill.md` has the required frontmatter fields and the required body sections — mirror the existing `TestEmbeddedFS_ObserverCoverageGapSkill` string assertions in `embedded_test.go`; do not invent new lint machinery).

## Success Criteria

- File `specs/observer-skills/prune-skill.md` exists.
- Frontmatter has the four fields `coverage-gap-skill.md` uses, and no others:
  - `id: prune` (kebab-case; this is the resolver key)
  - `description:` (one line describing the removal/consolidation lens)
  - `created:` (ISO-8601, UTC)
  - `priority:` (integer)
- Body contains, in this order, the section headings the coverage-gap template uses: `## Triggers`, `## Workflow`, `## Output Format`, `## Validation`, `## Examples`.
- The `## Workflow` defines exactly these four detection lenses, each described as evidence-backed and conservative:
  - **(a) Unused code** — code with **no caller, no test, and no reachable/reflected/generated reference** (genuinely dead). **Spec status is explicitly irrelevant to this lens** — the presence or absence of an assertion is not evidence either way.
  - **(b) Duplication → consolidation** — near-identical code that may point to a missing shared abstraction; framed as a **design candidate for a human**, not a prescribed merge.
  - **(c) Over-abstraction** — speculative generality (interfaces, options, layers) that has no current use; a design judgment surfaced for review.
  - **(d) Dead configuration** — flags, options, or config keys that are never read or exercised.
- The `## Workflow` explicitly states the skill is **recommend-only** and MUST NOT delete, edit, or move any code or spec — it only writes an observation (the observer's read-only contract, worded as in `coverage-gap-skill.md`).
- The `## Workflow` explicitly states that **the absence of a spec is never a signal to delete** — specs are progressive, so most code legitimately has no owning assertion, and un-spec'd code must not be flagged on that basis. It distinguishes `prune` from `coverage-gap`: `coverage-gap` encourages *documenting* used-but-unspec'd code, whereas `prune` flags code that is genuinely *unused* or redundant *by design*; the two overlap only at truly dead code.
- The `## Workflow` frames deletion and consolidation as **architecture/design decisions the skill only surfaces** for a human — not mechanical outputs.
- The `## Workflow` states a **bias toward NOT flagging**: when any plausible reason to keep the code exists (public/exported API surface, a caller the scan may not have resolved, reflection/plugin/generated usage, or a recent `coverage-gap` finding), the region is omitted, not flagged.
- The `## Output Format` follows the Observation Output Contract: writes to `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`; frontmatter includes `skill: prune`, `type: prune_candidate`, `severity: low|medium|high`, `affected_specs`, `affected_files`, `id`, `created`; body sections are `## Issue Description`, `## Evidence`, `## Impact`, `## Recommendation` in that order. Because the file lands under `observations/prune/`, `consolidate` and `DIGEST.md` pick it up automatically — no consolidate code change.
- The `## Validation` section asserts: output path is `observations/prune/...`; frontmatter uses `type: prune_candidate`; the skill wrote no code/spec files; Evidence lists concrete file paths (and caller / test / assertion absence) — no vague claims.
- The skill mirrors `coverage-gap`'s exclusions: `*_test.go`, generated code, and thin `cmd/` entry points are out of scope.
