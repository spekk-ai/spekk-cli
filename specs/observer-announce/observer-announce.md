---
id: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
---

# `spekk observer announce` — Deterministic Slack Announcement Step

## Overview

Announcing findings to Slack moves from prose instructions in a prompt to a
deterministic Go subcommand: `spekk observer announce`. It runs after the
scan/consolidate passes (cron-friendly, headless-safe), picks at most one
observation worth a human's attention, opens a Slack conversation for it via
the existing `spekk conversation open` mechanism, and records success as a
frontmatter commit on the observation's branch.

This is the direct fix for the production silent failure: announce eligibility
is computed from declarative state (branches + frontmatter, via the index),
not from comparing a digest against itself, and the announce behavior ships in
the binary, not in a deletable prompt file.

## Behavior

One invocation, in order:

1. `git fetch` (the only remote read).
2. Compute the top unannounced open observation from the index: severity
   `high` or `medium` only — `low` never announces; oldest-first (`created`)
   within severity.
3. For that observation's `observer/<slug>` branch, run the equivalent of
   `spekk conversation open` with:
   - title = the finding's title
   - body = a 2–3 sentence evidence summary, then
     `Proposed fix in PR: <url> — merge to accept, close to dismiss. Reply
     here to discuss.`, plus a severity warning
4. On success: commit the `announced:` frontmatter flip to the observer
   branch and push.

Hard caps enforced in code, not prose:

- At most **one** announcement per invocation.
- Evidence gate: no `affected` paths → refuse to announce.
- On failure: append to `.spekk/observer-conversation.log` and exit non-zero
  **without** the frontmatter flip (so the retry rule in
  `specs/observation-lifecycle/assertions/announced-marker-retry.md` re-sends
  next run).

## Sandbox Constraint

`spekk conversation open` only works inside a sandbox session — it writes to
the per-session spool directory named by `SPEKK_CONVERSATION_SPOOL`
(`specs/sandbox-conversation-open/`). `spekk observer announce` inherits that
constraint: outside a sandbox session (env var unset) it cannot deliver, and
must fail loudly rather than pretend to announce.

## Open Questions

- **`pr:` may be absent when announce runs** (the PR might not be opened yet,
  or PR-opening may be manual). Proposed default: announce anyway with the
  branch name in place of the PR URL ("Proposed fix on branch
  `observer/<slug>`"), since the branch — not the PR — is the state carrier.
  Alternative: treat a missing `pr:` like a missing evidence gate and skip.
  Builder should confirm with William before implementing.
- **Unpushed local observer branches:** if the selected observation's branch
  exists only locally, should announce push it first, skip it, or fail? The
  fetch-only rule covers reads; the success path already pushes the
  frontmatter flip, so pushing the branch is consistent — but this is not yet
  settled.

## Assertions

See `assertions/` for what must be true.
