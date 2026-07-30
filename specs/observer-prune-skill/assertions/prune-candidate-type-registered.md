---
id: prune-candidate-type-registered
parent: observer-prune-skill
created: 2026-07-25T12:00:00Z
priority: 1
status: done
branch: feature/observer-prune-skill
---

# `prune_candidate` Type Registered In Both Contract Docs

## Description

The Observation Output Contract lists the allowed `type` values in **two** places, and both must stay in sync. The `prune` skill introduces exactly one new type, `prune_candidate`, which must appear in both allowed-type lists, attributed to the `prune` skill in the same style the lists use for `coverage_gap`.

**Tests:** convention/string check — a test (or existing doc-parity check, if the repo has one) verifying the string `prune_candidate` appears in both `specs/observer-agent/observer.prompt.md` and `specs/observer-skill-discovery/observer-skill-discovery.md`. Do not add heavy machinery if no parity-check pattern already exists — a simple grep-style test is sufficient.

## Success Criteria

- The allowed-`type` list in `specs/observer-agent/observer.prompt.md` (the "Allowed `type` values" block) includes a `prune_candidate` entry, attributed to the prune skill (e.g. `prune_candidate` — prune skill (deletion / consolidation candidates)).
- The allowed-`type` list in `specs/observer-skill-discovery/observer-skill-discovery.md` includes the same `prune_candidate` entry, attributed the same way.
- Exactly one new type is introduced — `prune_candidate`. No `deletion_candidate` / `consolidation_candidate` split is added; sub-kinds live in the observation body, matching how `coverage_gap` is a single type.
- No existing type entries (`code_spec_misalignment`, `outdated_specs`, `compression_opportunity`, `spec_conflicts`, `coverage_gap`) are removed or renamed.
- The type name used here matches the `type:` value in the prune skill's Output Format (`prune-skill-markdown-exists`) — a single spelling, `prune_candidate`, across the skill and both contract docs.
