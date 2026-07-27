---
id: consolidate
description: Reviews every open observation across the visible observer/* branch union and curates the set through frontmatter status flips on the observations' own branches
created: 2026-07-11T14:00:00Z
priority: 2
---

# Consolidate

Keeps the observation set lean and trustworthy. Observations live on
`observer/<slug>` branches (`specs/observation-lifecycle/`); this skill is
the curation pass that reads all of them and prunes noise. Its output is
**frontmatter edits on the observations' own branches** — never a summary
artifact. `observations/DIGEST.md` is abolished: the digest is a rendered
view (`spekk observer digest`), a query over open observations,
severity-ranked, capped at 5.

## Triggers

- "consolidate"
- "prune observations"
- "clean up observations"
- "summarize observations"
- One-shot consolidation passes

## Scoped File-Write Exception

This skill may commit **observation frontmatter edits on `observer/<slug>`
branches** — that is its job. It never commits to main, never edits code or
specs, and never writes any digest or ledger file.

## Workflow

**CONTRACT: You must read every open observation before making any curation
decision. Concluding "nothing to prune" without completing the full review
is a contract violation.**

1. **Fetch, then discover.**
   - Run `git fetch` so remote-tracking `observer/*` branches are current
     (the only remote read).
   - Enumerate every observation across the visible branch union:
     `spekk observer digest --json` shows the ranked open set; read each
     `observer/<slug>` branch's `observations/<slug>.md` for the full record.

2. **Read every open observation in full.**
   - Extract: `slug`, `type`, `severity`, `status`, `created`, `affected`,
     and the body's Issue Description.
   - Do not skip, sample, or summarise before reading — partial reads
     constitute a contract violation.

3. **Identify duplicates.**
   - Two observations are duplicates when they share the same `type` and
     substantially the same `affected` set (exact-path overlap ≥ 50 %).
   - Keep the oldest as the canonical record; flip the newer duplicate(s) to
     `status: dismissed` on their own branches, noting the canonical slug in
     the commit message.

4. **Identify no-longer-worth-attention observations.**
   - An observation whose drift has demonstrably vanished (all `affected`
     paths gone or realigned), or that is stale (created > 30 days ago with
     no supporting evidence left), is a dismissal candidate.
   - Flip `status: open` → `dismissed` in its frontmatter, commit on its
     `observer/<slug>` branch, and push.
   - Leave the branch itself alone: deleting is a human decision (deletion
     invites re-flagging; parking suppresses).

5. **Never touch `announced:`** — that field belongs to
   `spekk observer announce`.

6. **Print a console summary** (a few lines only): observations reviewed,
   dismissed count, open count remaining per `spekk observer digest`.

7. **Exit** — do not start the default monitoring loop.

## Validation

- Every open observation across the visible branch union was read before any
  curation decision
- Duplicate observations (same `type`, ≥ 50 % `affected` overlap) are
  dismissed via frontmatter flips rather than left as parallel entries
- All curation output is observation frontmatter edits committed to
  `observer/<slug>` branches; no file outside those branches is written
- No committed digest, summary, or ledger artifact exists after the run —
  `observations/DIGEST.md` is never created
- Console output is a few summary lines; it does not dump observation bodies
- Skill exits after one pass; it does not enter the monitoring loop

## Examples

### Example 1: Duplicate findings across two branches

```
$ spekk observer consolidate
> Fetching... 6 observer branches visible
> Read 6 observations
> Duplicate: observer/parser-skips-drafts duplicates observer/parser-drops-draft-status
> Dismissed 1 observation (frontmatter flip pushed to its branch)
> Open observations: 5 (see `spekk observer digest`)
```

### Example 2: Clean repository, nothing to prune

```
$ spekk observer consolidate
> Fetching... 2 observer branches visible
> Read 2 observations
> Duplicates: 0, stale: 0 — nothing to curate
> Open observations: 2 (see `spekk observer digest`)
```
