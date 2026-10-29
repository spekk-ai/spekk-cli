---
slug: born-on-branches-omits-new-type-remedies
type: outdated_specs
severity: low
status: resolved
created: 2026-10-29T12:00:00Z
affected:
  - specs/observation-lifecycle/assertions/observations-born-on-branches.md
  - specs/observation-lifecycle/observation-lifecycle.md
  - specs/observer-agent/observer.prompt.md
---

# The born-on-branches assertion gives a remedy for only two of the four observation types

## Issue Description
Commit 2490022 (#329) made the lifecycle type set four types: `code_spec_misalignment`, `outdated_specs`, `underspecified_spec` and `missing_capability`. It stated the remedy for the two new types in the parent spec and in the observer prompt. It did not update the `observations-born-on-branches` assertion. That assertion says the remedy commit's shape "depends on `type`" and then lists only the two original types. It is still `status: done`, so the testable contract for the remedy commit has no criterion for half of the type set.

## Evidence
- `specs/observation-lifecycle/assertions/observations-born-on-branches.md` lines 27–30: "The remedy commit's shape depends on `type`:" followed only by `outdated_specs` (spec edit) and `code_spec_misalignment` (status flip).
- `specs/observation-lifecycle/observation-lifecycle.md` lines 63 and 75 (added in #329): the closed four-type set, and "`underspecified_spec` or `missing_capability`: a spec edit that states the intended behavior or capability".
- `specs/observer-agent/observer.prompt.md` lines 91–92 (added in #329): "`underspecified_spec`: a spec edit that states the intended result." and "`missing_capability`: a spec edit that states the missing capability."

## Impact
The parent spec and the prompt now say more than the assertion that is meant to verify them. A reviewer or builder checking the remedy shape against the assertion has no rule for an `underspecified_spec` or `missing_capability` remedy. A future prompt edit could drop those lines without failing any criterion. The type set and the remedy rules can then drift apart again without anyone noticing.

## Recommendation
Add the two new types to the assertion's remedy list, using the wording the parent spec already has (the remedy commit on this branch does this). No code change is needed.
