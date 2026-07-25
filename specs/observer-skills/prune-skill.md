---
id: prune
description: Recommend-only scan that surfaces deletion, consolidation, over-abstraction, and dead-config candidates for human review — the removal counterpart to coverage-gap
created: 2026-07-25T12:00:00Z
priority: 2
---

# Prune

Finds code and configuration that may be safe to remove, consolidate, or simplify. Where `coverage-gap` finds code with no spec backing and recommends documenting it, `prune` looks at the same kind of orphan signal and asks the opposite question: should this be deleted instead? `prune` never deletes anything itself — it is **recommend-only**; a human always makes the call.

## Triggers

- "prune"
- "what can we delete"
- "dead code"
- "duplication"
- "unused flags"
- "simplify the codebase"
- One-shot removal/consolidation scans

## Workflow

**This skill is recommend-only.** It MUST NOT delete, edit, or move any code or spec file — the observer's read-only contract holds even here; the only file this skill ever writes is a new observation under `observations/prune/`.

**Deletion advice is dangerous, so bias toward NOT flagging.** When any plausible reason to keep a region exists, omit it rather than flag it:
- It is part of a public/exported API surface
- A caller may exist that this scan didn't resolve (dynamic dispatch, reflection, plugins, generated call sites)
- It was added recently (check recent commit history / `created` dates) and may be an intentional, not-yet-used extension point
- A recent `coverage-gap` observation already covers the same files — defer to it rather than duplicate

Scan for exactly four lenses, each evidence-backed:

1. **(a) Deletion candidate** — a code region with **no owning assertion, no caller, and no test**. This is the removal counterpart to `coverage-gap`'s "document it" recommendation: `coverage-gap` finds the same orphan signal (code with no spec) and suggests writing a spec; `prune` finds it and suggests deleting the code instead. The two skills key off the same signal but reach opposite conclusions — the human decides document-vs-remove. Do not re-report a region already captured by a *recent* `coverage-gap` observation on the same files; treat "recent" as a judgment call, not a fixed window.
2. **(b) Duplication** — near-identical code (functions, blocks, config) that should be consolidated into a single source of truth.
3. **(c) Over-abstraction** — speculative generality: interfaces, options, or layers of indirection with no current caller, or only a single trivial implementation.
4. **(d) Dead configuration** — flags, options, or config keys that are never read or exercised by any code path.

Steps:
1. Scan `internal/` (and relevant config/flag definitions) for candidate regions under the four lenses above.
2. Cross-reference `specs/` (assertions, spec bodies) the same way `coverage-gap` does, to establish "no owning assertion."
3. Cross-reference callers within the codebase (search for references/usages) to establish "no caller."
4. Cross-reference `*_test.go` files to establish "no test."
5. Check `observations/coverage-gap/` for a recent observation touching the same files; if found, skip the finding or note the cross-reference instead of duplicating it.
6. Exclude `*_test.go`, generated code, and thin `cmd/` entry points from analysis — mirrors `coverage-gap`'s exclusions.
7. For each surviving candidate, assess severity:
   - **High** — duplicated or dead code with clear, concrete evidence of no callers/tests/specs across an entire region (e.g., a whole unused package)
   - **Medium** — a function/type/flag with no callers/tests/specs inside an otherwise-used package
   - **Low** — minor duplication or a single unused option among otherwise-used config
8. Write a single consolidated observation file to `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`.
9. Print a console summary: total candidates, breakdown by lens and severity, path to the full report.
10. Exit after one scan — do not enter the default monitoring loop.

## Output Format

The observation file follows the shared Observation Output Contract:

```yaml
---
id: prune-{timestamp-slug}
created: 2026-07-25T15:30:00Z
skill: prune
type: prune_candidate
severity: high | medium | low    # max across findings
affected_specs: []               # specs that reference (or fail to reference) the flagged code
affected_files:
  - internal/path/to/candidate.go
---

# Prune Report

## Issue Description
N regions of code/config are candidates for deletion, consolidation, or simplification.

## Evidence
- `internal/legacy/oldthing.go`: `OldThing()` — no assertion references it, no caller found, no test exercises it (lens a: deletion candidate)
- `internal/a/foo.go` / `internal/b/bar.go`: near-identical 40-line blocks, candidate for consolidation (lens b: duplication)
- (... per-candidate list, one bullet per finding, naming the lens and the concrete evidence ...)

## Impact
Unused and duplicated surface adds maintenance cost and reading burden without
adding value; dead config can silently diverge from what's actually exercised.
Left alone, this grows over time since nothing else in the loop is charged with
finding it.

## Recommendation
For each finding, one of:
- Delete the code/config (a human decision — this skill does not delete)
- Consolidate duplicates into a single implementation
- If a `coverage-gap` observation already suggests documenting the same region, flag the conflict for a human to resolve (document vs remove)
```

## Validation

- Output file exists at `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`
- Frontmatter includes `type: prune_candidate` (and the other required Output Contract fields: `id`, `created`, `skill: prune`, `severity`, `affected_specs`, `affected_files`)
- Body contains all four required sections: Issue Description / Evidence / Impact / Recommendation
- Evidence lists concrete file paths and states which of caller / test / assertion is absent — no vague claims
- Skill wrote no files outside `observations/prune/` — no code or spec file was created, edited, or deleted
- Test files (`*_test.go`), generated code, and thin `cmd/` entry points are excluded from analysis
- Every finding maps to exactly one of the four lenses (a)-(d)
- No finding duplicates a *recent* `coverage-gap` observation on the same files without cross-referencing it
- Skill exits after one scan; does not start the monitoring loop

## Examples

### Example 1: First-time scan finds a mix of lenses
```
$ spekk observer prune
> Indexing specs/ ... 45 specs, 178 assertions
> Scanning internal/ ... 23 packages
> Cross-referencing callers, tests, coverage-gap observations ...
> Found 5 candidates: 1 high, 2 medium, 2 low (2 deletion, 2 duplication, 1 dead config)
> Report: observations/prune/2026-07-25T15-30-00Z.md
```

### Example 2: Clean, actively-maintained codebase
```
$ spekk observer prune
> Indexing specs/ ... 60 specs, 240 assertions
> Scanning internal/ ... 25 packages
> Cross-referencing callers, tests, coverage-gap observations ...
> Found 0 candidates — bias toward not flagging kept borderline cases out
> Report: observations/prune/2026-07-25T15-32-00Z.md
```
