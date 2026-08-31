---
id: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
---

# Observation Lifecycle — Branches as State, Frontmatter as Record

## Overview

Observations become declarative repo artifacts whose lifecycle is readable
purely from git. An observation is a markdown file under `observations/` with
YAML frontmatter; it is born on a dedicated `observer/<slug>` branch together
with a proposed remedy; the set of observer branches **is** the state machine.
No prompt-maintained ledgers, no committed digest file, no GitHub API calls to
determine state.

## Motivation

The observer → Slack announce pipeline in a production sandbox deployment
failed silently for three days. Two root causes, both prompt-driven-state
failures:

1. **Self-referential dedup.** Announce logic compared the digest against
   itself: the scan pass rewrote `observations/DIGEST.md` before the
   consolidate pass read it, so "new since previous digest" was never true and
   nothing ever announced.
2. **Untracked prompt state.** The prompt extension carrying the announce
   instructions lived as an untracked file in the sandbox clone and was
   deleted by unrelated git hygiene within 48 hours, silently removing the
   behavior.

Conclusion: **state must live declaratively in the repo** (git branches + YAML
frontmatter). The SQLite index (`.spekk/index.db`, `internal/index`) is
strictly a derived/ephemeral layer (see `specs/observation-index/`), and
announce mechanics move from prose to a deterministic Go subcommand (see
`specs/observer-announce/`).

## Design

### Observation file format

An observation is a markdown file under `observations/` with frontmatter:

```yaml
---
slug: parser-drops-draft-status        # kebab-case, matches branch name
type: code_spec_misalignment           # code_spec_misalignment | outdated_specs
severity: high                         # high | medium | low
status: open                           # open | resolved | dismissed
created: 2026-07-26T12:00:00Z
announced: 2026-07-26T13:05:00Z        # absent until a Slack conversation opened
pr: https://github.com/org/repo/pull/7 # optional
affected:                              # evidence — required, non-empty
  - internal/parser/parser.go
  - specs/spec-validation/assertions/draft-excluded.md
---
```

**No evidence, no observation** — an observation without at least one
`affected` path is invalid.

### Birth: branches, not main

The observer NEVER commits observations to main. Each finding is born on a
branch named `observer/<slug>` containing two SEPARATE commits:

1. The observation file (status `open`).
2. The proposed remedy:
   - spec-side drift (`outdated_specs`): the actual spec edit
   - code-side drift (`code_spec_misalignment`): only the assertion status
     flip `done` → `failed` — no code changes

Separate commits let a reviewer take one without the other.

### The branch set is the state machine

State is readable purely via git; `git fetch` is the only remote call. PR
open/closed status is deliberately invisible and irrelevant — **no GitHub API
calls for state**.

| Git fact | Lifecycle state |
| --- | --- |
| `observer/<slug>` visible locally or on origin | announced / pending |
| branch merged to main | resolved (observation lands on main with `status: resolved`, remedy applied in the same merge) |
| branch kept, its PR closed | **parked** — still in the union, still deduped, never re-announced |
| branch deleted | **forgotten** — union forgets it; persistent drift is legitimately re-found by the next scan |

Human convention (stated here and in every observer-generated PR body):
**close-without-delete = parked; delete = fair game to re-flag.**

Dedup at scan time uses the cross-branch union machinery
(`internal/crossbranch`, the same mechanism as `spekk show`'s merge-preview
mode): a scan must not re-flag drift a **live claim** already covers — the
observation read from the branch named after it, whose slug has not reached
main. An inherited copy is not a claim, because every branch is cut from
main and carries a copy of everything already merged. See
`specs/only-a-live-claim-covers/`.

### Announce marker and idempotent retry

After a Slack conversation opens successfully, one more commit on the observer
branch sets `announced:` in the frontmatter. Retry rule: branch exists but
frontmatter lacks `announced:` → send the Slack pointer. This makes announce
recovery idempotent (see `specs/observer-announce/`).

### Digest is a view, not a file

`observations/DIGEST.md` and any announce ledger file are abolished. The
digest becomes a rendered view — a query over open observations, capped at 5,
ranked by severity — never a committed artifact. The existing consolidate
skill's curation becomes frontmatter edits (status flips), not maintenance of
a summary file.

## Notes

- **Conflict surfacing:** two observer branches editing the same spec lines
  surface as conflicts via `spekk show`'s cross-branch mode, as it already
  does (`specs/show-cross-branch/`). Confirmed-conflict detection requires
  git >= 2.38 (`git merge-tree --write-tree`); older git degrades gracefully
  to the coarser classification, as already specced there.
- **Supersedes interim contract:** this replaces an interim prompt-only
  announce contract deployed to a client repo (`.spekk/observer.prompt.md`
  with an `observations/announced.log` ledger). That deployment should be
  simplified to match once these specs ship.
- **Supersedes prior digest direction:** the DIGEST.md-as-surface direction in
  `specs/observer-agent/` (assertions `consolidation-skill-exists`,
  `digest-as-default-surface`) and `specs/observer-skills/consolidate-skill.md`
  is superseded where it conflicts with this spec; the consolidate skill's
  *curation judgment* survives, its *output artifact* does not.

## Open Questions

- **Who flips `status: open` → `resolved` at merge?** Proposed default: the
  merge convention is that the accepting reviewer (or a merge-time automation)
  flips it; the indexer may additionally derive effective status (present on
  main ⇒ not open) so a lagging frontmatter cannot make main appear to carry
  open observations. Left open for the builder/reviewer to settle.
- **Slug reuse after "forgotten":** when a deleted branch's drift is re-found,
  does the new observation reuse the same slug (and thus branch name) or get a
  dated suffix? Reuse is simpler; a suffix preserves history if the old branch
  is ever restored.

## Assertions

See `assertions/` for what must be true.
