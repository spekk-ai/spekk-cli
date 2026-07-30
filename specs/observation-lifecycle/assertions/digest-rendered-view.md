---
id: digest-rendered-view
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 2
status: done
depends-on: observation-file-format
---

# The Digest Is a Rendered View Over Open Observations, Not a Committed Artifact

## Description

`observations/DIGEST.md` is abolished as a committed file. The digest becomes
a query: the open observations, severity-ranked, capped at 5. The consolidate
skill's curation survives as frontmatter edits (status flips), not as
maintenance of a summary file.

## Success Criteria

- No part of the observer workflow (prompt, skills, Go subcommands) writes or
  requires a committed `observations/DIGEST.md`. The observer prompt and the
  consolidate skill contain no instruction to maintain a digest file.
- The digest is defined as a rendered view with these exact semantics: all
  observations with `status: open` across the visible branch union, ranked by
  severity (`high` > `medium` > `low`), capped at 5 entries.
- The consolidate skill's curation actions are expressed as observation
  frontmatter edits — e.g. flipping `status: open` → `dismissed` on the
  observation's own branch — never as edits to a summary artifact.
- The prior digest-based assertions
  (`specs/observer-agent/assertions/digest-as-default-surface.md` and the
  digest-maintenance portions of
  `specs/observer-skills/consolidate-skill.md`) are updated or superseded so
  no live spec instructs maintaining DIGEST.md.

**Note:** "rendered view" deliberately does not mandate a delivery surface.
Candidates: a `spekk query` invocation documented in the observer prompt, a
`spekk observer digest` subcommand, or a panel in `spekk show`. See the
parent spec's open questions — the builder should pick the cheapest surface
that satisfies the semantics above and record the choice here.

**Decision (recorded):** the surface is the `spekk observer digest`
subcommand (with `--json` for tooling). It renders directly from the
cross-branch union (`internal/observation`, `Union.Digest`), so it needs no
index freshness dance and encodes the exact semantics — open only, main
presence excluded as a backstop, severity-ranked, oldest first within a
severity, capped at 5 — in one tested code path. A documented `spekk query`
invocation was rejected because the semantics (per-slug dedup, the on-main
backstop) would live in copy-pasted SQL; a `spekk show` panel was rejected
as strictly more expensive.

**Tests:** internal/observation/observation_test.go (TestDigestSemantics),
cmd/spekk/observer_test.go (TestObserverDigestOutput)
