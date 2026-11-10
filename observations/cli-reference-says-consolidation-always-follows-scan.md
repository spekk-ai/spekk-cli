---
slug: cli-reference-says-consolidation-always-follows-scan
type: code_spec_misalignment
severity: medium
status: open
created: 2026-11-10T12:00:00Z
affected:
  - docs/cli-reference.md
  - internal/agent/observer_cron.go
  - specs/observer-agent/assertions/install-cron.md
---

# CLI reference says consolidation always runs 30 minutes after the scan

## Issue Description

The `install-cron` assertion was revised in #350. It now says that the
consolidation entry moves to minute 30 only for intervals of an hour or
longer. Sub-hourly intervals keep their `*/N` form with no offset, and a
sub-hourly scan can still start together with consolidation. The code does
exactly this. The user-facing CLI reference still says, without condition,
that consolidation "runs 30 minutes after the scan". The doc comment on
`consolidateCron` also contradicts the function body below it. Both have one
cause: the prose around the schedule was not updated when the offset was
extended from the daily case to every hourly-form interval.

## Evidence

- `docs/cli-reference.md:691` (in `### spekk observer install-cron`):
  "The consolidation entry runs 30 minutes after the scan, so it curates what
  the scan filed." No condition is stated.
- `specs/observer-agent/assertions/install-cron.md`, success criteria:
  "For any interval of an hour or longer the consolidation entry is rendered
  at minute 30 … A sub-hourly scan that fires at minute 30 can still start
  together with consolidation. … Sub-hourly intervals keep their `*/N` form
  with no offset."
- `internal/agent/observer_cron.go`, `consolidateCron`: it returns
  `minutesToCron(minutes)` (that is, `*/N * * * *`) for sub-hourly intervals,
  and `30 …` only for multiples of 60. Its doc comment says "Below a day the
  schedules are intervals rather than times, so there is no hour to move; the
  pre-existing overlap there is unchanged". The body does move the 60-minute
  and 6-hour cases (`30 * * * *`, `30 */%d * * *`).
- With `--loop-interval 30 --consolidate-interval 30`, both lines render
  `*/30 * * * *` and fire at the same instant. The CLI reference tells the
  reader that this cannot happen.
- With `--loop-interval 1440 --consolidate-interval 60`, consolidation runs
  at :30 every hour. That is not "30 minutes after the scan" in any useful
  sense.

## Impact

An operator who reads the CLI reference and picks sub-hourly intervals
expects the two runs to be staggered. In fact they can start two headless
sessions together in one working tree, and the spec names that as the hazard
the offset exists to prevent. The stale doc comment points the next builder
at the old daily-only rule.

## Recommendation

Re-open `install-cron` for the builder. Reword the CLI reference sentence to
match the assertion: half past for intervals of an hour or longer, no offset
for sub-hourly intervals, which can still coincide. Correct the
`consolidateCron` doc comment so it describes the body. The schedule code
itself already matches the spec and needs no change.
