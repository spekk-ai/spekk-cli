---
id: observer-agent
created: 2026-07-11T14:00:00Z
priority: 2
---

# Observer Agent

## Overview

The observer is the quality-assurance layer of the spec-driven system: a
read-only agent that scans specs and code for drift and records what it finds
as observation files under `observations/`. Its prompt lives at
`specs/observer-agent/observer.prompt.md`; its skill system and the
observation output contract are specced in `specs/observer-skill-discovery/`.

This spec holds the observer's own assertions and documents the direction the
observer is heading, so that direction is not lost between working sessions.

## Vision: Curator, Not Firehose

**The problem observed in practice:** when observations were treated as
first-class artifacts to surface to users, the observer generated far too many
of them. Surfacing a raw stream of observations is overwhelming and trains
users to ignore the observer entirely. Detection is not the bottleneck —
curation is.

**The direction:** the observer becomes a two-stage system, modeled on the
heartbeat/consolidation pattern used in agent harnesses that manage
long-running memory:

1. **Raw observations are cheap and private.** Scan modes (default loop,
   skills) write freely to `observations/{mode}/` per the existing output
   contract. These are the observer's working memory — dated, verbose,
   never shown to users directly. Analogous to dated memory files in a
   heartbeat-driven agent.

2. **A consolidation pass maintains a lean curated surface.** A distinct
   consolidation mode periodically reviews all raw observations: merges
   duplicates, archives resolved or stale items, and distills what remains
   into a single capped digest ranked by severity. The digest is the *only*
   observer output users are expected to read. Analogous to the lean
   permanent memory a consolidation turn maintains by pruning aged-out content.

Principles carried over from agent heartbeat systems:

- **Silent suppression.** A scan with nothing worth saying produces nothing
  user-facing. The observer earns attention by speaking rarely.
- **Progressive disclosure.** The curated surface holds only a few items at a
  time. New items enter as old ones are resolved or archived — users are
  never handed twenty findings at once.
- **Forced deliberation.** Consolidation must actually re-read each open
  observation before concluding "nothing to prune." In real traces, models
  reflexively skip pruning when it is optional; the prompt must mandate the
  review, not suggest it.
- **Harness-owned scheduling (future).** When consolidation becomes
  scheduled rather than user-invoked, the spekk CLI — not the agent —
  decides when it runs, via a deterministic state marker (last-consolidated
  timestamp owned by Go, not inferred from filenames). The deeper reason
  to prefer Go-owned scheduling over cron is that it enables the agent to
  update its own task queue: the harness can hand the agent its next
  scheduled work item and let it self-direct, rather than relying on an
  external scheduler with no awareness of agent state.

## Non-Goals (For Now)

Explicitly out of scope until the consolidation loop proves itself through
manual use:

- **Sandbox integration** — running the observer loop inside spekk's cloud
  sandbox is a future concern; nothing here should be designed around it.
- **External harness as executor** — using an external agent harness to drive
  the observer loop is future work; only the heartbeat/consolidation *patterns* are adopted now.
- **Parser enforcement of observations** — already tracked as future work in
  `specs/observer-skill-discovery/observer-skill-discovery.md`. Consolidation
  should stabilize the digest format first.
- **Go-owned consolidation scheduling** — the state marker and interval logic
  wait until the skill is worth automating.

## Sequencing

The smallest meaningful first step is the consolidation skill itself
(`assertions/consolidation-skill-exists.md`): markdown-only, rides the
existing observer skill discovery, requires no Go changes, and directly
attacks the overwhelm problem. Each later stage is only justified by
experience with the previous one:

1. Consolidation skill, user-invoked (`spekk observer consolidate`) — now
2. Digest becomes the default surfacing path (prompt-level, quiet default loop)
3. Cron-based scheduling: `spekk observer install-cron` writes crontab entries for the observer loop and consolidation; Go-owned scheduling (with agent self-directed task queue) is a future upgrade
4. Parser-enforced observation/digest validation, `spekk` query commands
5. Sandbox / external-harness execution of the observer loop

## Assertions

See `assertions/` for what must be true about the observer agent.
