---
id: exit-on-complete
parent: builder-loop
created: 2026-07-09T16:00:00Z
priority: 1
status: done
depends-on: assertion-count-tracking
branch: feature/advanced-loop
---

# Exit on Complete

The builder loop exits cleanly when all assertions are complete, instead of polling forever.

## Success Criteria

- When `spekk next` returns `type: "complete"` and the loop has completed at least one assertion this run, the loop prints a summary and exits with code 0
- When `spekk next` returns `type: "complete"` on the first iteration (no work to do), the loop exits immediately with a "nothing to do" message and code 0
- A `--watch` flag opts into the current stay-alive behavior (poll every 5s waiting for new work) for users who want the loop to keep running indefinitely
- Default behavior (no `--watch`) is exit-on-complete
