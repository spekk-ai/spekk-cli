---
slug: agent-prompts-state-old-parent-status-rule
type: outdated_specs
severity: medium
status: resolved
created: 2026-10-18T00:00:00Z
announced: 2026-10-18T01:26:37Z
affected:
  - specs/coach-agent/coach.prompt.md
  - specs/builder-agent/builder.prompt.md
  - internal/parser/parser.go
  - specs/status-counts-match-the-work/assertions/spec-not-started-until-work-starts.md
  - README.md
  - docs/concepts.md
---

# Coach and builder prompts still state the old parent-status rule

## Issue Description

#281 changed how a spec's status is derived. A spec is now `not_started` until
one of its counted (not draft) assertions is `in_progress` or `done`. The coach
and builder prompts were not updated. Both still say that any incomplete child
makes the parent `in_progress`. That was the rule #276 reported as a bug. Both
prompts are embedded in the binary (`embedded.go:11`), so every coach and
builder session gets the outdated rule.

## Evidence

- `specs/status-counts-match-the-work/assertions/spec-not-started-until-work-starts.md`
  (status `done`): "All counted (not draft) assertions `not_started`: the spec
  is `not_started`."
- `internal/parser/parser.go:958-1003` (`computeParentStatus`): returns
  `in_progress` only when a counted child is `done` or `in_progress`; otherwise
  `not_started`.
- `specs/coach-agent/coach.prompt.md:486`: "If any child is incomplete → parent
  becomes `in_progress`".
- `specs/builder-agent/builder.prompt.md:184`: the same line. The builder list
  also leaves out the draft exclusion and the `not_started` cases.

## Impact

An agent that reasons from its prompt expects a spec it has just written, with
only `not_started` assertions, to show as 🚧. `spekk status` now shows 📋. The
agent may report the difference as a bug or "correct" it. The prompts and the
parser describe two different rules.

## Recommendation

Update both prompt lists to match `computeParentStatus`. The remedy commit on
this branch makes that edit.
