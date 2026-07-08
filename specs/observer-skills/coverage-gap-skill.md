---
id: coverage-gap
description: Inverse-drift scan that finds implementation code in internal/ with no spec or assertion referencing it
created: 2026-05-22T12:00:00Z
priority: 2
---

# Coverage Gap

Finds implementation code that has no spec backing it. The default observer loop checks **spec → code** (does the code match what specs declare?). This skill inverts the lens and checks **code → spec** (does every piece of code have an assertion that mentions it?).

## Triggers

- "coverage gap"
- "code without specs"
- "unbacked code"
- "find untracked code"
- "what code has no spec"
- One-shot inverse drift scans

## Workflow

1. Scan `internal/` for code regions worth checking: Go packages, exported types, exported functions
2. Build an index of every spec/assertion reference to file paths, package names, type names, and function names from `specs/`
3. For each code region, check whether any spec/assertion references it (by package, by file path, or by name)
4. Group unbacked regions by package — note which packages have partial vs. zero spec coverage
5. Skip regions explicitly excluded: test files (`*_test.go`), generated code, `cmd/` entry points (often thin)
6. Assess severity:
   - **High** — entire package has no spec coverage
   - **Medium** — exported API (types, functions) with no spec coverage in an otherwise-spec'd package
   - **Low** — internal helpers with no spec coverage in a spec'd package
7. Write a single consolidated observation file to `observations/coverage-gap/YYYY-MM-DDTHH-MM-SSZ.md`
8. Print a console summary: total unbacked regions, breakdown by severity, path to the full report
9. Exit after one scan — do not enter the default monitoring loop

## Output Format

The observation file follows the observer output contract:

```yaml
---
id: coverage-gap-{timestamp-slug}
created: 2026-05-22T15:30:00Z
skill: coverage-gap
type: coverage_gap
severity: high | medium | low    # max across findings
affected_specs: []               # specs that *should* cover this code but don't
affected_files:
  - internal/path/to/unbacked.go
---

# Coverage Gap Report

## Issue Description
N code regions in `internal/` have no spec or assertion referencing them.

## Evidence
- `internal/sandbox/`: entire package, no spec references this directory or its exports
- `internal/serve/server.go`: exported `Serve()` function, no assertion mentions it
- (... per-region list with file paths and what's unbacked ...)

## Impact
Code without spec backing can drift silently — changes don't trigger any spec
re-validation, and the spec-driven loop has no signal that this code exists.
Either the code is genuinely undocumented requirements (write a spec) or it's
dead code (remove it).

## Recommendation
For each high-severity finding, either:
- Create a spec describing what the code must be true about, OR
- Confirm the code is intentional and add an assertion that declares its existence
For medium/low findings, batch into a "coverage cleanup" spec.
```

## Validation

- Output file exists at `observations/coverage-gap/YYYY-MM-DDTHH-MM-SSZ.md`
- Frontmatter includes all required fields: `id`, `created`, `skill: coverage-gap`, `type: coverage_gap`, `severity`, `affected_specs`, `affected_files`
- Body contains all four required sections: Issue Description / Evidence / Impact / Recommendation
- Evidence section lists concrete file paths or package names — no vague claims
- Test files, generated files, and entry points (`cmd/`) are excluded from analysis
- Skill does not modify any code or spec files — only writes the observation
- Skill exits after one scan; does not start the monitoring loop
- Console summary count matches the findings in the written observation

## Examples

### Example 1: First-time scan on a partially-spec'd codebase
```
$ spekk observer coverage-gap
> Indexing specs/ ... 45 specs, 178 assertions
> Scanning internal/ ... 23 packages
> Cross-referencing ...
> Found 12 unbacked regions: 2 high, 4 medium, 6 low
> Report: observations/coverage-gap/2026-05-22T15-30-00Z.md
```

### Example 2: Well-covered codebase
```
$ spekk observer coverage-gap
> Indexing specs/ ... 60 specs, 240 assertions
> Scanning internal/ ... 25 packages
> Cross-referencing ...
> Found 1 unbacked region: 0 high, 0 medium, 1 low
> Report: observations/coverage-gap/2026-05-22T15-32-00Z.md
```
