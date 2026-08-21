---
id: spec-validation
created: 2026-07-23T21:10:51Z
priority: 1
---

# `spekk validate` — a hard gate for spec invariants

## Problem

The builder's lock and status protocol is prompt-only cooperation. The
"CRITICAL LOCK RULES" and status sections in `builder.prompt.md` (`done` /
`failed` / `not_started` / `draft` must carry no `locked-by`; status must be a
valid enum; priority 1–3;
`depends-on` must resolve; parent specs carry no rolled-up status) are enforced
by nothing in code. On a solo project a careful author holds these in their
head. With strangers and CI, they need a machine gate.

Separately, the working-tree parser (`internal/parser`) is deliberately
**lenient**: `spekk next` / `spekk list` skip a malformed spec or assertion file
with a stderr warning and keep going, and duplicate assertion IDs are warned and
skipped. That is correct for a work queue (one typo shouldn't stall everyone),
but it means a broken assertion silently *vanishes* from the queue instead of
failing loudly. There is no strict mode that says "everything under specs/ is
well-formed."

## Solution

A new `spekk validate` command that checks the currently-prose invariants and
exits non-zero on any violation, so it is usable identically by a human at the
terminal, by CI, and by the builder loop before it commits.

`validate` is **strict where `next`/`list` are lenient**: any spec/assertion
file that the parser would skip-with-warning is instead a hard failure.

## Invariants enforced (see `validate-command` for the authoritative list)

1. Every `.md` file under `specs/*/` that has YAML frontmatter parses cleanly
   (valid kebab-case `id`, ISO-8601 `created`, `priority` 1–3, valid `status`
   enum, required fields present). No silent skips.
2. Every assertion's `parent` names an existing spec.
3. Every `depends-on` names an existing assertion; no self-reference; no cycles.
4. No duplicate assertion IDs and no duplicate spec IDs.
5. Lock state: only `in_progress` may carry a `locked-by`, and it need not
   carry one. See the `lock-is-a-live-claim` spec for why the reverse
   requirement was removed.
6. Parent spec files carry no rolled-up status (field absent, or `draft` only —
   the one value the parser honors on a parent).

## Scope

- In scope: the `validate` command, its invariant checks, exit-code contract,
  human-readable output, and a builder-prompt instruction to run it before
  committing.
- Out of scope (deliberately, per the lean mandate): rule-configuration, severity
  levels, a generic YAML-schema engine, auto-fix, git-hook scaffolding, JSON
  output, and any validation of the observer's Observation Output Contract (a
  separate future spec). None of these belong in v1.

## Design decisions to sanity-check

- **`draft` is the one legal parent status.** The coach doctrine says parent
  specs have no status field, and today none do. But the parser *does* honor
  `status: draft` on a parent to exclude a whole spec from the queue. Forbidding
  every status value except `draft` catches the real footgun (e.g. `status: done`
  on a parent, which the parser silently overwrites) without breaking that
  working escape hatch.
- **Reuse `internal/parser`, don't fork it.** The exported `ParseSpecContent` /
  `ParseAssertionContent` wrappers already run the full per-file frontmatter
  validation; `validate` drives those per file to surface every error, rather
  than reimplementing frontmatter parsing.
