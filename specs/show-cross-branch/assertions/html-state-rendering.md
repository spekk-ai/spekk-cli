---
id: html-state-rendering
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 2
status: done
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
- The inline cross-branch state badge (in the spec tree and as the spec-level
  summary) is **icon-only**: a single compact glyph per state — incoming-addition
  `+`, incoming-modification a change glyph, conflict `⚠`, incoming-deletion `✕`,
  plus the dashed/unconfirmed-conflict variant — with **no** redundant text label
  beside it (e.g. it shows `⚠`, not `⚠ Conflict`, and `+`, not `+ Incoming add`).
  The human-readable label is still discoverable: it is exposed as a `title`
  tooltip on the inline badge and shown in full only in the detail panel's
  per-branch contribution list, where there is room for words.
- Each spec shows a **spec-level summary badge** derived from the rollup field,
  so conflicts are visible without expanding every assertion.
- Every spec/assertion — including **foreign incoming-addition** items that exist
  only on another branch — renders complete, legible priority and status badges
  from its real metadata. A cross-branch item must **never** show an empty or
  box-shadow-only status badge or a blank/zero priority badge, and row-level
  treatments (reduced opacity, background tint) must not render the priority or
  status badges illegible.
- The contributing branch name(s) are visible for each cross-branch item (e.g. on
  hover or inline), since the same item may have contributions from multiple
  branches.
- When git is in degraded mode, unconfirmed/potential conflicts are visually
  distinguished from confirmed conflicts and the degraded-mode notice is shown.
- Unchanged specs/assertions render exactly as they do today; cross-branch styling
  applies only to items with contributions.
