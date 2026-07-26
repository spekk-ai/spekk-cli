---
id: pr-body-conventions
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 2
status: not_started
depends-on: observations-born-on-branches
---

# Observer-Generated PR Bodies State the Merge/Close/Delete Convention

## Description

Because PR status is invisible to the tooling, the humans reviewing observer
PRs carry part of the state machine. Every observer-generated PR body must
therefore teach the convention at the point of decision.

## Success Criteria

- Every PR the observer opens for an `observer/<slug>` branch includes, in its
  body, the three outcomes in plain language:
  - **merge** — accept the finding and its remedy; the observation lands on
    main as resolved
  - **close without deleting the branch** — park it: the finding stays
    suppressed and will not be re-announced
  - **delete the branch** — forget it: the observer is free to re-flag the
    drift if it persists
- The same convention is documented in the parent spec
  (`specs/observation-lifecycle/observation-lifecycle.md`) as the canonical
  statement, and the PR-body text is derived from it.
- The observer prompt (or the Go code that opens the PR, if PR opening is
  automated) is the single place the PR-body template lives, so the wording
  cannot drift per-finding.

**Note:** for permanent dismissal (drift is real but accepted), the PR body
should point at the `.spekk/dont-flag.yaml` mechanism
(`specs/observer-dont-flag/`) rather than suggesting branch deletion, which
would only lead to re-flagging.
