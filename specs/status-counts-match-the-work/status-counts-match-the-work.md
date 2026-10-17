---
id: status-counts-match-the-work
created: 2026-10-17T00:00:00Z
priority: 2
---

# Status counts match the work

## Problem

`spekk status` shows two wrong values:

- A spec where no counted assertion has started shows as in progress (🚧). `computeParentStatus` returns `in_progress` for any spec that is not all done (issue #276).
- A draft assertion is in the `done/total` count of its spec and in the overall total, but not in the spec status. A spec with 6 done and 1 draft shows `✅ ... (6/7 assertions complete)`, and a reader thinks that it is not finished (issue #274).

## Solution

`computeParentStatus` returns `not_started` when all counted (not draft) assertions are `not_started`. The rules for `failed`, `done` and draft stay the same. `spekk status` leaves draft assertions out of the totals and shows them as a separate count.
