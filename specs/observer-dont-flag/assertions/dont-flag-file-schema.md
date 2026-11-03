---
id: dont-flag-file-schema
parent: observer-dont-flag
created: 2026-07-26T12:00:00Z
priority: 1
status: done
kind: invariant
---

# `.spekk/dont-flag.yaml` Has a Defined, Validated Entry Schema

## Description

The suppression file is a YAML list at `.spekk/dont-flag.yaml` on main. Each
entry has a required match pattern, a required reason, a required author, and
an optional expiry date. Anything that reads the file validates entries and
fails loudly on malformed ones.

## Success Criteria

- The documented entry schema is exactly:
  - `match` — required; a path glob (matched against `affected` paths of a
    would-be observation) or an observation slug pattern
  - `reason` — required, non-empty string
  - `by` — required, non-empty string
  - `until` — optional date; when absent the entry is permanent
- The file is tracked in git at `.spekk/dont-flag.yaml` (repo root `.spekk/`,
  alongside — but unlike — the gitignored `index.db`). It is committed on
  main; nothing about the design requires it on other branches.
- A missing file means "no suppressions" and is not an error.
- A malformed file or an entry missing `match`, `reason`, or `by` causes the
  consuming tool to exit non-zero with a message naming the offending entry —
  suppressions are safety-relevant, so a broken file must not be silently
  treated as empty (which would cause a re-flag flood) or silently skipped.
- An entry whose `until` date has passed no longer suppresses anything.

## Decisions

- `until` is a date-only value, read as end of day UTC. The entry suppresses through the whole named day and expires at the next UTC midnight. Expiry is silent in v1 (no warnings).
- An unknown entry field is an error. It is not ignored, because a typo such as `untill` that spekk drops with no message turns a suppression that must expire into a permanent one.

## Rationale

`reason` and `by` are required because they are the point of the mechanism. A suppression with no owner and no reason is the invisible prompt-side state that this design removes.

**Tests:** internal/dontflag/dontflag_test.go (TestParseValidFile,
TestParseRejectsMalformedEntries, TestUntilExpiresEndOfDayUTC),
cmd/spekk/observer_test.go (TestScanCheckMalformedDontFlagFailsLoudly)
