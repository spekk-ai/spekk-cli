---
id: html-state-rendering
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 2
status: done
branch: feat/show-cross-branch
depends-on: cross-branch-data-model
---

# The Explorer Visually Distinguishes the Four Cross-Branch States

## Description

`internal/show/template.html` renders the cross-branch contributions so a user
can see, at a glance, which specs/assertions are incoming additions, clean
modifications, conflicts, or deletions — and which branch(es) each came from.

## Success Criteria

- New CSS classes / icon treatments distinguish the four states, added alongside
  the existing `.status-*` classes and the `statusEmoji` JS map in
  `template.html`:
  - **incoming-addition** — reduced opacity or background tint + a distinct icon.
  - **incoming-modification (clean)** — a distinct treatment that highlights
    changed state, including assertion **status drift**.
  - **conflict** — explicitly and prominently called out (it must not look like a
    normal item).
  - **incoming-deletion** — its own distinct treatment.
- Each spec shows a **spec-level summary badge** derived from the rollup field,
  so conflicts are visible without expanding every assertion.
- The contributing branch name(s) are visible for each cross-branch item (e.g. on
  hover or inline), since the same item may have contributions from multiple
  branches.
- When git is in degraded mode, unconfirmed/potential conflicts are visually
  distinguished from confirmed conflicts and the degraded-mode notice is shown.
- Unchanged specs/assertions render exactly as they do today; cross-branch styling
  applies only to items with contributions.
