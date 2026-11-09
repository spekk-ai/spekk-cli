---
slug: install-cron-spec-says-consolidation-hour-later
type: outdated_specs
severity: medium
status: open
created: 2026-11-09T00:00:00Z
affected:
  - specs/observer-agent/assertions/install-cron.md
  - internal/agent/observer_cron.go
  - internal/agent/observer_cron_test.go
  - docs/cli-reference.md
---

# install-cron Assertion Says Consolidation Runs an Hour After the Scan

## Issue Description

The `install-cron` assertion (status `done`) says that when both intervals
are daily, "the consolidation entry is rendered an hour later". The
implementation moves it **half an hour** later, and it does the same for
every hourly-form interval, not only the daily one. The code comment says why
an hour would be wrong: an hour later at minute 0 collides with an hourly scan
(`0 * * * *` fires at 01:00 too), while minute 30 misses every scan schedule
of an hour or longer. The CLI reference already documents the half-hour
offset. The assertion is the only text that still says an hour.

## Evidence

- `specs/observer-agent/assertions/install-cron.md` lines 32–36: "when both
  are daily the consolidation entry is rendered an hour later so the two runs
  do not start together".
- `internal/agent/observer_cron.go` `consolidateCron` (around lines 176–203):
  a daily interval returns `30 0 * * *`, a one-hour interval returns
  `30 * * * *`, and other multiples of 60 return `30 */H * * *`. The comment
  reads: "Half an hour is enough … An hour later at minute 0 would not: an
  hourly scan (`0 * * * *`) fires at 01:00 too."
- `internal/agent/observer_cron_test.go` around lines 300–310 asserts
  `30 0 * * *`, `30 */6 * * *` and `30 * * * *`.
- `docs/cli-reference.md` line 691: "The consolidation entry runs 30 minutes
  after the scan".

## Impact

A builder that re-reads this assertion as the contract could "fix"
`consolidateCron` to an hour later. That brings back the collision the code
avoids for an hourly scan schedule. The assertion also does not mention the
offset for hourly intervals, so that behaviour has no spec behind it.

## Recommendation

Update the assertion to describe what is built: consolidation is offset to
minute 30 for every interval of an hour or longer, including the daily
default (`30 0 * * *`), and sub-hourly intervals keep their `*/N` form with no
offset. The remedy commit on this branch makes that edit.
