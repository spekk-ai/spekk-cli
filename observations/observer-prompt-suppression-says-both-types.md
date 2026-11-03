---
slug: observer-prompt-suppression-says-both-types
type: outdated_specs
severity: low
status: resolved
created: 2026-11-02T12:00:00Z
affected:
  - specs/observer-agent/observer.prompt.md
  - internal/dontflag/dontflag.go
  - internal/observation/observation.go
---

# Observer prompt says a suppression closes a path "for both types"

## Issue Description

#329 grew the closed lifecycle type set from two types to four
(`underspecified_spec` and `missing_capability` were added). Most of the
observer prompt was updated, but the dedup rule in step 1 still says a
`dont-flag` suppression "closes it for both types". There are now four types.
A suppression ignores the type completely, so the sentence understates what
it closes.

## Evidence

- `specs/observer-agent/observer.prompt.md` line 179: "A suppression is the
  one thing that closes a path itself, and it closes it for both types — a
  `dont-flag` match does not check the observation type."
- `internal/observation/observation.go` line 117: `allowedTypes` lists four
  types.
- `internal/dontflag/dontflag.go` line 102: `Suppressed(entries, slug,
  affected, now)` takes no type, so a match suppresses an observation of any
  of the four types.
- Elsewhere in the same prompt (lines 41, 69 and 195), the type set is
  "the four closed lifecycle types".

## Impact

The prompt contradicts itself. An observer that reads "both types" as
"the two drift types" could conclude that a suppressed path is still open to
an `underspecified_spec` or `missing_capability` finding. It would then waste
a scan building a finding that `scan-check` suppresses. Behavior is not
affected, because `scan-check` is the authority.

## Recommendation

Change "for both types" to "for every type" in the prompt. The remedy commit
on this branch makes that one-word change.
