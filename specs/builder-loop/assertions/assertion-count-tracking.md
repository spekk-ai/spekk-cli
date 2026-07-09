---
id: assertion-count-tracking
parent: builder-loop
created: 2026-07-09T16:00:00Z
priority: 1
status: not_started
branch: feature/advanced-loop
---

# Assertion Count Tracking

The builder loop tracks the number of assertions completed during a run and reports the count when the loop ends.

## Success Criteria

- A counter increments each time a builder agent exits successfully (exit code 0) for an assertion
- When the loop ends (all complete or interrupted), the final log line includes the count: e.g. `"Builder loop complete. 5 assertions completed."`
- When interrupted via Ctrl+C, the count is still reported before exit
- When no assertions were completed (nothing to do), the message reflects that: `"No assertions to work on."`
