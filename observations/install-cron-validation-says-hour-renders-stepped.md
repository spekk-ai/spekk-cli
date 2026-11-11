---
slug: install-cron-validation-says-hour-renders-stepped
type: outdated_specs
severity: medium
status: resolved
created: 2026-11-11T00:00:00Z
announced: 2026-11-11T20:02:40Z
affected:
  - specs/observer-agent/assertions/install-cron.md
  - internal/agent/observer_cron.go
---

# install-cron validation section says 60 renders as `*/60` and has no upper bound

## Issue Description
The "Interval validation rejects non-cron-expressible values" section of the
install-cron assertion still describes the rules as first written. It says an
interval `<= 60` renders as a sub-hourly `*/N * * * *` and that any exact
multiple of 60 renders as `0 */H * * *`. The code renders 60 as `0 * * * *`
and 1440 as `0 0 * * *`, and refuses anything above 1440. The same assertion's
own success criteria already say a day renders as `0 0 * * *` and that longer
intervals are refused, so the assertion contradicts itself.

## Evidence
- `specs/observer-agent/assertions/install-cron.md`, about lines 126-134: "accepted
  only if it is either `<= 60` (rendered as a sub-hourly `*/N * * * *`) or an
  exact multiple of 60 (rendered as an hourly `0 */H * * *`)". It also keeps
  "today `minutesToCron(90)` yields `*/90 * * * *`", which is no longer true
  because 90 is refused at parse time, plus the garbled example "45-when->60".
- The same file, about lines 46-50: "a day renders as `0 0 * * *` and anything
  longer is refused at parse time."
- `internal/agent/observer_cron.go`, `minutesToCron`: 60 returns `0 * * * *`
  and 1440 returns `0 0 * * *`. Its doc comment explains that `*/60` and
  `*/24` are rejected by strict crons. `isValidCronInterval` returns false
  above `maxCronInterval` (1440).

## Impact
A builder that follows the validation section literally would render 60 as
`*/60 * * * *`, the expression that `minutesToCron` exists to avoid, and would
accept multiples of 60 above a day. The assertion is `done`, so nothing
re-checks it, and the contradiction stays in place for the next reader.

## Recommendation
Rewrite the validation section to match the code: an interval is accepted if
it is below 60, exactly 60, or an exact multiple of 60 up to 1440. Below 60
renders as `*/N`, 60 as `0 * * * *`, 1440 as `0 0 * * *`, and the rest as
`0 */H * * *`. Remove the stale "today" sentence and the garbled example. The
remedy commit on this branch makes that edit.