---
id: consolidation-skill-exists
parent: observer-agent
created: 2026-07-11T14:00:00Z
priority: 2
status: not_started
---

# An Observer Consolidation Skill Exists

## Description

A `consolidate` observer skill exists as a package skill so that
`spekk observer consolidate` reviews all raw observations and maintains a
single lean, curated digest — the only observer output users are expected to
read. This is the first concrete step of the curator-not-firehose vision in
the parent spec: raw observations stay cheap and private; the digest stays
small and trustworthy.

## Success Criteria

- `specs/observer-skills/consolidate-skill.md` exists, follows the same
  structure as the seed `coverage-gap` skill (frontmatter, Triggers,
  Workflow, Validation, Examples), and resolves via
  `spekk skill show observer consolidate`
- `specs/observer-skills/consolidate-skill.md` is listed in the `//go:embed`
  directive in `embedded.go` (alongside `coverage-gap-skill.md`) so that
  `spekk observer consolidate` resolves the skill from the embedded
  filesystem on a `go install`ed binary — not only when the source tree is
  present. Without this, an installed binary silently falls back to the
  default observer loop instead of running the consolidate skill
- The skill's workflow mandates reading every open observation file across
  all `observations/*/` subdirectories before any pruning decision —
  concluding "nothing to prune" without the review is a contract violation
  (forced deliberation)
- The skill merges duplicate observations and archives resolved or stale
  ones by moving them to `observations/archive/` (originals preserved, not
  deleted)
- The skill maintains a curated digest at `observations/DIGEST.md` holding
  at most 5 open items, ranked by severity, each linking to its underlying
  raw observation file(s)
- Updating the digest is mandatory on every run — even a quiet run rewrites
  it (possibly unchanged, possibly empty) so it always reflects the latest
  consolidation, and the run's console output stays to a few summary lines
- The skill documents its scoped exception to the per-mode output rule: it
  may move and rewrite files anywhere under `observations/` (that is its
  job), but still never writes outside `observations/` — code and specs
  remain untouched
- The skill exits after one consolidation pass; it does not start the
  monitoring loop
