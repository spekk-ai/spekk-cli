---
id: consolidation-skill-exists
parent: observer-agent
created: 2026-07-11T14:00:00Z
priority: 2
status: done
---

# An Observer Consolidation Skill Exists

## Description

A `consolidate` observer skill exists as a package skill so that
`spekk observer consolidate` reviews every open observation and curates the
set. This is the curator-not-firehose vision in the parent spec: the open
observation set stays small and trustworthy.

**Superseded in part** by
`specs/observation-lifecycle/assertions/digest-rendered-view.md`: the skill's
curation judgment survives, but its output artifact does not — curation is
expressed as observation frontmatter edits (status flips on `observer/<slug>`
branches), and the digest is a rendered view (`spekk observer digest`), never
a committed `observations/DIGEST.md`.

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
- The skill's workflow mandates reading every open observation across the
  visible `observer/*` branch union before any curation decision —
  concluding "nothing to prune" without the review is a contract violation
  (forced deliberation)
- The skill curates through observation frontmatter edits on the
  observations' own branches: duplicates and no-longer-relevant findings are
  flipped to `status: dismissed`, never deleted and never moved to an
  archive directory
- The skill maintains no committed digest, summary, or ledger artifact; the
  at-most-5, severity-ranked view is the rendered `spekk observer digest`
- The run's console output stays to a few summary lines
- The skill documents its scoped write exception: observation frontmatter
  edits on `observer/<slug>` branches only — code, specs, and main remain
  untouched
- The skill exits after one consolidation pass; it does not start the
  monitoring loop
