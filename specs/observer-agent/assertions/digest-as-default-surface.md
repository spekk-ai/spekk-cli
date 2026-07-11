---
id: digest-as-default-surface
parent: observer-agent
created: 2026-07-11T15:00:00Z
priority: 2
status: not_started
depends-on: consolidation-skill-exists
---

# The Digest Is the Observer's Default Surface

## Description

The observer's default loop closes each scan cycle with a quiet consolidation
pass and reports only from `observations/DIGEST.md` — never the raw
observation stream. This is stage 2 of the curator-not-firehose vision in the
parent spec: raw observations remain the observer's private working memory,
and the curated digest becomes the single surface users see. This is a
prompt-level change to `specs/observer-agent/observer.prompt.md`; no Go
changes are required.

## Success Criteria

- `specs/observer-agent/observer.prompt.md` instructs the default loop to
  keep writing raw observations to `observations/default/` per the existing
  output contract — the raw stream is unchanged by this stage
- The prompt mandates that each scan cycle ends with a consolidation pass
  using the same logic as the `consolidate` skill (merge duplicates, archive
  resolved/stale items, rewrite `observations/DIGEST.md`)
- Console output to the user is limited to a brief summary drawn from
  `DIGEST.md` — the number of open items and their severities — and never
  prints raw observation text
- Silent suppression: if `DIGEST.md` is empty or does not exist yet, the
  agent says nothing to the user for that cycle
- The `consolidate` skill remains separately invocable via
  `spekk observer consolidate`; this change only makes consolidation happen
  automatically at the end of each default loop cycle
