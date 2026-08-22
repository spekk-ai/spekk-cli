---
id: lock-is-a-live-claim
created: 2026-08-21T17:35:19Z
priority: 1
---

# A lock says a builder holds an assertion now, not that the assertion is in progress

## Problem

Two rules contradict each other, and a coach that follows the documented one writes a tree that `spekk validate` rejects (GitHub issue #193).

`specs/coach-agent/coach.prompt.md` tells the coach to move an edited `done` or `failed` assertion to `in_progress`, so the builder re-implements it. `internal/validate/validate.go:236` then fails that file: `status is in_progress but locked-by is missing`. A coach cannot supply the value. A lock names a builder session, and no `lock`, `unlock`, `claim`, or `release` command exists in the CLI.

So the coach has three choices and all three are wrong: leave the tree failing validation, invent a lock, or ignore its own rule.

**The invented lock is the worst of the three.** `validate` checks only that the field is present. A coach-invented value and a lock left by a builder that crashed have the same shape, so a reader cannot tell them apart. And a stale lock is invisible: `validate` flags a lock on a `done` assertion, and a missing lock on an `in_progress` one, but never a lock that is months old, because that shape is legal. On a tree of a few hundred assertions the reporter found locks weeks old while validation reported the tree clean.

## The code already disagrees with the validator

`parser.IsLockStale` (`internal/parser/parser.go:867`) reads a unix timestamp from the tail of the `locked-by` value and calls a lock stale after two hours. `FindNextAssertion` (line 941) skips an `in_progress` assertion only when its lock is **fresh**, and `IsLockStale("")` returns true. So `spekk next` already treats an unlocked `in_progress` assertion as free work. Only `validate` insists the lock must be there.

That settles which rule is the outlier.

## Solution

Treat the lock as the answer to "does a builder hold this right now", not as part of the definition of `in_progress`.

1. `validate` accepts `in_progress` with no `locked-by`. An assertion somebody started and nobody holds is a real state, and it is the state a crashed builder leaves behind.
2. `validate` reports a stale lock, which the old rule made unreportable because the shape was legal.
3. The coach prompt writes `not_started` for an edited `done` or `failed` assertion. That puts the assertion back in the work queue, which is the stated intent, and it needs no lock the coach cannot mint.

## Scope

- In scope: the `validate` lock invariant, a stale-lock report, and the coach prompt's status rule.
- Out of scope, deliberately: a `needs_rebuild` status value. It would ripple through the list filters, the parent-status computation, the index schema, the docs, and both agent prompts, and `not_started` already means "in the queue, not yet built". Also out of scope: a `lock` or `release` CLI command, and any configurable staleness threshold. `parser.IsLockStale` already fixes two hours, and one threshold is enough.

## Design decisions to sanity-check

- **The reverse rule stays.** A `done`, `failed`, `not_started`, or `draft` assertion that carries a `locked-by` is still a failure. That direction catches a real mistake and nothing legitimate produces it.
- **A stale lock is a warning, not a failure.** A builder crash is not a broken spec tree, and `spekk next` already recovers from it by ignoring the stale lock. Failing CI over it would punish the wrong person.
