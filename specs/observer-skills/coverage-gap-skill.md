---
id: coverage-gap
description: Optional scan that surfaces implementation code a spec could usefully document — a progressive-adoption aid, not a defect report. Un-spec'd code is normal, never flagged as a problem.
created: 2026-05-22T12:00:00Z
priority: 2
---

# Coverage Gap

Surfaces implementation code that no spec currently documents, as **optional suggestions** for where adding a spec might be worthwhile. The default observer loop checks **spec → code** (does the code match what specs declare?). This skill looks the other way — **code → spec** — and asks: *is there code embodying an invariant the team would benefit from documenting?*

**Un-spec'd code is normal, not a defect.** spekk specs are *progressive*: adoption is encouraged but always incomplete, so the vast majority of code legitimately has no owning assertion, and that is the expected steady state — not drift, not an error, not a gap that must be filled. This skill only points at the *few* places where writing a spec would genuinely add value; leaving the rest un-spec'd is a fine outcome. And it never comments on whether code should be **removed** — that is a separate, usage-based judgment that belongs to the `prune` skill, not this one.

## Triggers

- "coverage gap"
- "code without specs"
- "what code might be worth documenting"
- "where would a spec add value"
- One-shot code→spec documentation-opportunity scans

## Workflow

1. Scan `internal/` for code regions worth considering: Go packages, exported types, exported functions
2. Build an index of every spec/assertion reference to file paths, package names, type names, and function names from `specs/`
3. For each code region, check whether any spec/assertion references it (by package, by file path, or by name)
4. Skip regions explicitly excluded: test files (`*_test.go`), generated code, `cmd/` entry points (often thin)
5. From the un-referenced regions, surface only the ones where a spec would plausibly add value — code that carries a **behavioral invariant, contract, or constraint the team would want protected**. Do NOT surface code merely because it lacks a spec; most of it should be left alone.
6. Assess severity by **documentation value, never by how much is un-spec'd**:
   - **Medium** — code embodying an important, currently-undocumented invariant that a spec would meaningfully protect (rare)
   - **Low** — a suggestion a human might find useful; easy to decline
   - (There is no "high": the absence of a spec is never a critical finding.)
7. Write a single consolidated observation file to `observations/coverage-gap/YYYY-MM-DDTHH-MM-SSZ.md`
8. Print a console summary: number of documentation opportunities surfaced, path to the full report
9. Exit after one scan — do not enter the default monitoring loop

## Output Format

The observation file follows the observer output contract:

```yaml
---
id: coverage-gap-{timestamp-slug}
created: 2026-05-22T15:30:00Z
skill: coverage-gap
type: coverage_gap
severity: medium | low           # never high — spec-absence is not a critical finding
affected_specs: []               # an existing spec this code could optionally extend, if any — NOT "specs that should exist"
affected_files:
  - internal/path/to/candidate.go
---

# Coverage Gap Report

## Issue Description
N code regions where adding a spec could be worthwhile — optional documentation suggestions for a human to consider. (Un-spec'd code that isn't listed here is fine as-is; this is not a coverage scorecard.)

## Evidence
- `internal/serve/server.go`: `Serve()` implements the request lifecycle — an invariant a spec could usefully pin down (documentation opportunity)
- (... per-region list; each names the invariant a spec would protect, not merely "has no spec" ...)

## Impact
Documenting a load-bearing invariant makes it explicit and lets the spec-driven
loop protect it from silent change. This is an OPPORTUNITY, not a problem: code
without a spec is the normal state of a progressively-adopted repo, not drift.
Whether a given region is worth documenting is a judgment call for a human.

## Recommendation
Each item is an OPTIONAL suggestion:
- If the code embodies an invariant worth protecting, consider adding (or extending) a spec.
- Otherwise, leaving it un-spec'd is a perfectly good outcome — decline freely.
Never treat un-spec'd code as cleanup debt, never add a spec just to "declare existence,"
and never recommend removing code because it lacks a spec (that's `prune`'s usage-based call).
```

## Validation

- Output file exists at `observations/coverage-gap/YYYY-MM-DDTHH-MM-SSZ.md`
- Frontmatter includes all required fields: `id`, `created`, `skill: coverage-gap`, `type: coverage_gap`, `severity`, `affected_specs`, `affected_files`
- `severity` is never `high` — the absence of a spec is not a critical finding
- Body contains all four required sections: Issue Description / Evidence / Impact / Recommendation
- Every surfaced region names the **invariant a spec would protect**, not merely "has no spec"
- Findings are framed as OPTIONAL documentation opportunities, never as defects, gaps-to-fill, drift, or cleanup obligations
- No finding recommends removing code, or adding a spec purely to declare existence
- Test files, generated files, and entry points (`cmd/`) are excluded from analysis
- Skill does not modify any code or spec files — only writes the observation
- Skill exits after one scan; does not start the monitoring loop

## Examples

### Example 1: A few genuine documentation opportunities
```
$ spekk observer coverage-gap
> Indexing specs/ ... 45 specs, 178 assertions
> Scanning internal/ ... 23 packages
> Surfacing code where a spec would add value ...
> 2 documentation opportunities (0 high, 1 medium, 1 low) — most un-spec'd code left alone
> Report: observations/coverage-gap/2026-05-22T15-30-00Z.md
```

### Example 2: Nothing worth flagging
```
$ spekk observer coverage-gap
> Indexing specs/ ... 60 specs, 240 assertions
> Scanning internal/ ... 25 packages
> Surfacing code where a spec would add value ...
> 0 opportunities — un-spec'd code here is fine as-is; nothing pressing to document
> Report: observations/coverage-gap/2026-05-22T15-32-00Z.md
```
