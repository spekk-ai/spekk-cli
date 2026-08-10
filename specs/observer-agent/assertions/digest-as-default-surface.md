---
id: digest-as-default-surface
parent: observer-agent
created: 2026-07-11T15:00:00Z
priority: 2
status: done
depends-on: consolidation-skill-exists
---

# The Digest Is the Observer's Default Surface (Superseded: Rendered View, Not a File)

## Description

**Superseded where it concerned `observations/DIGEST.md`** by
`specs/observation-lifecycle/assertions/digest-rendered-view.md`: the digest
is now a rendered view — a query over open observations across the visible
branch union, severity-ranked, capped at 5 — surfaced by
`spekk observer digest`, never a committed file. The part of this assertion
that survives is the surface discipline: the observer's console output stays
a brief digest summary, never the raw observation stream.

## Success Criteria

- `specs/observer-agent/observer.prompt.md` contains no instruction to write
  or maintain `observations/DIGEST.md` (or any committed digest artifact)
- The prompt directs reporting through the rendered view: run
  `spekk observer digest` and print at most a one-line summary of its output
- Silent suppression: when the rendered digest is empty, the agent says
  nothing to the user for that cycle
- Console output never prints raw observation text
- The `consolidate` skill remains separately invocable via
  `spekk observer consolidate`; its curation is expressed as observation
  frontmatter edits on observer branches (see
  `specs/observer-skills/consolidate-skill.md`)
