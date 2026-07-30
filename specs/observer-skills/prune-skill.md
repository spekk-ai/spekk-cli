---
id: prune
description: Recommend-only scan that surfaces genuinely-unused code and design-level redundancy (duplication, over-abstraction, dead config) as candidates for human architectural review — never triggered by the absence of a spec
created: 2026-07-25T12:00:00Z
priority: 2
---

# Prune

Surfaces code and configuration that a human might choose to remove or consolidate — genuinely unused code, and design-level redundancy — as candidates for review. It is **recommend-only**: it writes observations, never deletes or edits anything. Every finding is an **input to a human's architecture and design decision**, not a directive.

**The absence of a spec is never a reason to flag code.** Specs in spekk are *progressive* — adoption is encouraged but always incomplete, and the vast majority of code legitimately has no owning assertion. `prune` reasons about **usage and design**, not spec coverage. (Used-but-undocumented code is a `coverage-gap` concern — *add a spec* — not a `prune` one. The two only meet at genuinely dead code, which is worth neither documenting nor keeping.)

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

**Deletion and consolidation are architecture/design decisions, not mechanical outputs.** The skill's job is to *surface* candidates with concrete evidence and hand the judgment to a human. Bias toward NOT flagging — when any plausible reason to keep a region exists, omit it:

- **No spec is not a signal.** Never flag code merely because no assertion references it — that is the normal state of a progressively-adopted codebase, not a defect.
- It is part of a public/exported API surface, or a documented extension point.
- A caller may exist that this scan didn't resolve — dynamic dispatch, reflection, plugins, generated call sites, build tags.
- It was added recently (check commit history / `created` dates) and may be intentional, not-yet-wired work.

Scan for four lenses, each evidence-backed:

1. **(a) Unused code** — a code region with **no caller, no test, and no reachable/reflected/generated reference**: genuinely dead. **Spec status is irrelevant to this lens** — do not use the presence or absence of an assertion as evidence either way. Report it as a *candidate for removal* for a human to confirm is truly unreachable.
2. **(b) Duplication → consolidation** — near-identical code that may point to a **missing shared abstraction**. This is a *design* observation: surface the duplication and the concept it suggests, but state plainly that whether (and how) to consolidate is an architecture call for a human — do not prescribe a merge.
3. **(c) Over-abstraction** — indirection (interfaces, options, layers) with no current caller, or only a single trivial implementation: speculative generality a human may want to collapse. A design judgment, surfaced for review.
4. **(d) Dead configuration** — flags, options, or config keys that are never read or exercised by any code path.

Steps:
1. Scan `internal/` (and relevant config/flag definitions) for candidate regions under the four lenses above.
2. Establish "no caller": search the codebase for references/usages, accounting for dynamic and generated call sites where you can, and being explicit in the evidence about what you could and couldn't resolve.
3. Establish "no test": cross-reference `*_test.go` files.
4. Exclude from analysis: `*_test.go`, generated code, thin `cmd/` entry points, and any public/exported API surface or documented extension point.
5. For each surviving candidate, assess severity:
   - **High** — a whole region (e.g. an entire package) with concrete evidence of no callers, tests, or references.
   - **Medium** — a function/type/flag with no callers or tests inside an otherwise-used package; or duplication that clearly wants a single home.
   - **Low** — minor duplication, a single unused option, or a borderline over-abstraction.
6. Write a single consolidated observation file to `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`.
7. Print a console summary: total candidates, breakdown by lens and severity, path to the full report.
8. Exit after one scan — do not enter the default monitoring loop.

## Output Format

The observation file follows the shared Observation Output Contract:

```yaml
---
id: prune-{timestamp-slug}
created: 2026-07-25T15:30:00Z
skill: prune
type: prune_candidate
severity: high | medium | low    # max across findings
affected_specs: []               # usually empty — NEVER populated merely because code lacks a spec;
                                 # list a spec only if a finding genuinely references one (e.g. a spec points at now-dead code)
affected_files:
  - internal/path/to/candidate.go
---

# Prune Report

## Issue Description
N regions of code/config are candidates for removal or design-level consolidation, for a human to review.

## Evidence
- `internal/legacy/oldthing.go`: `OldThing()` — no caller found, no test exercises it, no reachable/reflected reference (lens a: unused code). [scan could not rule out reflection — noted]
- `internal/a/foo.go` / `internal/b/bar.go`: near-identical 40-line blocks — may point to a shared helper; consolidation is a design call (lens b: duplication)
- (... one bullet per finding, naming the lens and the concrete usage/design evidence — never spec-absence ...)

## Impact
Genuinely unused surface and design-level redundancy add maintenance cost and
reading burden. Left alone they grow, since nothing else in the loop is charged
with finding them. (This is about code that isn't used or is redundant by
design — not about code that merely lacks a spec.)

## Recommendation
Each finding is an input to a human's design decision, e.g.:
- Confirm the code is truly unreachable, then remove it.
- Consider whether the duplication warrants introducing a shared abstraction — an architecture call, not a mechanical merge.
- Collapse the unused indirection if the generality isn't earning its keep.
Never recommend removing code on the grounds that it has no spec.
```

## Validation

- Output file exists at `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`
- Frontmatter includes `type: prune_candidate` (and the other required Output Contract fields: `id`, `created`, `skill: prune`, `severity`, `affected_specs`, `affected_files`)
- Body contains all four required sections: Issue Description / Evidence / Impact / Recommendation
- Every deletion (lens a) finding cites **no caller AND no test AND no reachable reference** — and cites usage evidence, never spec-absence
- `affected_specs` is empty unless a finding genuinely references a spec — it is NEVER populated merely because code is un-spec'd
- Consolidation/over-abstraction findings (lenses b, c) are framed as design candidates for a human, not prescribed merges
- Skill wrote no files outside `observations/prune/` — no code or spec file was created, edited, or deleted
- `*_test.go`, generated code, thin `cmd/` entry points, and public API/extension points are excluded from analysis
- Every finding maps to exactly one of the four lenses (a)-(d)
- Skill exits after one scan; does not start the monitoring loop

## Examples

### Example 1: First-time scan finds a mix of lenses
```
$ spekk observer prune
> Scanning internal/ ... 23 packages
> Resolving callers, tests, references ...
> Found 5 candidates: 1 high, 2 medium, 2 low (2 unused, 2 duplication, 1 dead config)
> Report: observations/prune/2026-07-25T15-30-00Z.md
```

### Example 2: Clean, actively-maintained codebase
```
$ spekk observer prune
> Scanning internal/ ... 25 packages
> Resolving callers, tests, references ...
> Found 0 candidates — un-spec'd but used code was left alone; borderline cases omitted
> Report: observations/prune/2026-07-25T15-32-00Z.md
```
