---
id: only-a-live-claim-covers
created: 2026-08-28T12:00:00Z
priority: 1
---

# Only a Live Claim Covers a Finding

## Problem

`FindCovering` in `internal/observation/union.go` decides whether a new finding is already filed. It accepts any observation at any ref that is not main, and it matches on the type plus **any one** overlapping `affected` path. Both halves of that test are wrong, and together they hide real drift.

### An inherited copy is not a claim

A merged observation lives on main forever. Every `observer/*` branch is cut from `origin/main`, so every such branch carries an identical copy of it, and that copy is at a branch ref. `!isMainRef(o.Ref)` therefore passes, and the copy covers. One open observer branch is enough to make all of resolved history cover at once.

```
$ git checkout -b observer/unrelated main
$ spekk observer scan-check --type code_spec_misalignment --slug new \
    --affected internal/parser/parser.go
{"result":"covered","slug":"old-drift","branch":"observer/old-drift",
 "ref":"refs/heads/observer/unrelated"}
```

The suppression does not drain. An open finding closes a file only until its branch merges, but a resolved one lives on main forever and is re-carried by every branch created after it. The ratchet runs the wrong way: the more drift a team fixes in a file, the more permanently that file is closed to new findings, and the busiest files accumulate resolutions fastest.

### A shared file is not the same finding

Two unrelated findings that name the same file collide. A resolved finding about tenant scoping suppressed an unrelated finding about placeholder columns, because the code for both lives in one handler file. The `affected` list of a `code_spec_misalignment` finding names the assertion **and** the code, so the code path alone is the half the two findings share.

### The reported branch is a guess

The `branch` field of a `covered` result is `BranchName(covering.Slug)`. It is rebuilt from the slug instead of read from the ref the observation was found at, so it can name a branch that does not exist — `observer/old-drift` above, whose branch was deleted at merge.

## Solution

An observation suppresses a new finding only while it is a **live claim**: somebody has filed this exact finding, its branch is still there, and the work is not finished.

1. **The ref must name the observation.** An observation counts for the branch it is named after — `observer/<slug>` carries `observations/<slug>.md`, the convention `BranchName` already encodes. An inherited copy sits at another finding's branch and never matches. A copy on main never matches either, so the rule replaces `isMainRef` rather than adding to it.
2. **Main ends the claim.** A slug present on main is resolved history, whatever its branch-local frontmatter says. This is the backstop the package already documents and `Digest` already applies, and it settles the merged-but-undeleted branch: main wins, so the finding is resolved while its branch survives.
3. **The match is the type plus the slug.** The slug is what the finding is called; an overlapping path is only where it lives.

## Why the slug and the branch, not `status: resolved`

The issue proposes the owning-branch rule, and a comment proposes a simpler one: `status: resolved` never covers. The resolved rule fixes the reported case, but it is the weaker of the two.

- **It reads a field that is allowed to lag.** The lifecycle makes the branch set the state machine and the frontmatter the record, and the package comment already names presence on main as the backstop for exactly this reason. A merge that lands without the status flip leaves an `open` copy on main, which every later branch inherits — the same permanent ratchet, with a different value in the file.
- **It ends a claim too early.** A branch whose PR flips the status to `resolved` before the merge is still the live claim on that drift. Under the resolved rule the file re-opens to a duplicate exactly while the remedy is in review, and the duplicate wants a branch name that is already taken.
- **The owning-branch rule needs no field at all.** An inherited copy is not a claim because of where it is, not because of what it says.

Presence on main gives the resolved semantics that the comment asks for, from git rather than from frontmatter, so both halves of the comment's intent survive.

## Scope

- In scope: `FindCovering`, `Covers`, the `branch` field of the `scan-check` result, the lifecycle package comment, and the observer prompt text that teaches the old dedup rule.
- Out of scope, deliberately:
  - **A configurable dedup key.** One rule the reader can state is worth more than a setting each repository tunes differently.
  - **A relation between findings** (supersedes, related-to, a recurrence counter). The union is a set of files in git, and a link field would need a writer, a validator, and a repair path.
  - **A dated slug for a slug an observer branch already holds.** `ResolveSlug` dates a slug taken on main only. A candidate that keeps the plain slug while a branch of that name exists is a pre-existing hazard, and it is not what this issue reports.
  - **Path overlap as a second, weaker match.** Keeping it as a fallback keeps the false negative it causes.
  - **A warning for an observation whose slug does not match its branch.** Such a file stops being a claim, and the branch it should have been on is free, so the next scan files the finding properly. A new warning channel for a filing that the prompt already forbids costs more than it saves.

## Cost

Dedup gets narrower, so the same drift re-found under a different name is filed twice. That is the trade the package already makes where it refuses to reduce a directory to the files under it: a duplicate is visible and a person can close it, and a false negative hides real drift and nobody learns of it.

## Assertions

See `assertions/` for what must be true.
