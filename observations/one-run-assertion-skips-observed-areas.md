---
slug: one-run-assertion-skips-observed-areas
type: outdated_specs
severity: medium
status: resolved
created: 2026-11-08T12:00:00Z
announced: 2026-11-08T20:03:40Z
affected:
  - specs/observer-agent/assertions/one-observation-per-run.md
  - specs/observer-agent/observer.prompt.md
---

# One-observation-per-run assertion still says a run skips areas that carry an observation

## Issue Description

The `done` assertion `one-observation-per-run` says that the area choice of a
scan "skips areas that carry an open observation or a suppression, because
those cannot produce a filing." That is the rule that #205 removed. Dedup now
matches the type and the slug of a live claim, not a path. The observer prompt
says the opposite of the assertion: skip findings that are already filed,
never the files they name. A second, different finding in a file that an open
observation names comes back `clear`, and filing it is correct. A suppression
closes only the paths that it matches. It does not close an area.

## Evidence

- `specs/observer-agent/assertions/one-observation-per-run.md:42-47`: "...
  and it skips areas that carry an open observation or a suppression, because
  those cannot produce a filing."
- `specs/observer-agent/observer.prompt.md:179`: "**Skip findings that are
  already filed, never the files they name.** ... A second, different finding
  in a file that claim also names comes back `clear`, and filing it is right,
  because two findings in one file are two findings. A suppression is the one
  thing that closes a path itself ..."
- Commit 4fa0b79 (#205, "Let only a live claim suppress a new finding"): "The
  match is the type and the slug, not an overlapping affected path." That
  commit did not change the assertion.

## Impact

The assertion is `done`, so its criteria read as the contract of the prompt.
If a reviewer or the builder checks the prompt against this criterion, the
criterion asks for the area-skipping rule again. That rule would let one open
observation hide all later drift in the files that it names. #205 fixed that
ratchet. A reader who sees the two texts cannot tell which one is correct.

## Recommendation

Change the clause in the assertion so that it agrees with the prompt and with
#205. The area choice skips findings that are already filed (a live claim
with the same type and slug), not the files that they name. A suppression
closes only the paths that it matches. The remedy commit on this branch makes
this change.