---
id: status-flag-filters-assertions
parent: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 1
status: done
depends-on: full-hierarchy-without-flags
---

# `--status` Flag Filters Assertions by Status

## Description

`spekk list --status <value>` returns only assertions whose `status` field matches the given value, in the requested output format.

## Success Criteria

- `spekk list --status draft --json` returns a flat assertion list where every assertion has `"status": "draft"`.
- Assertions with a different status are absent from the output.
- The assertion status determines inclusion. The computed parent spec status has no effect on filtering.
- `spekk list --status done` includes only done assertions.
- `spekk list --status not_started` includes only not_started assertions.
- `spekk list --status in_progress` includes only in_progress assertions.
- `spekk list --status failed` includes only failed assertions.
- An invalid status value (e.g. `--status bogus`) causes the command to exit with a nonzero status and print a message listing the valid status values.
- A missing or empty status value causes a nonzero exit with an error that identifies `--status`, including when another flag follows it.
- Filtering does not affect `spekk next --all` or any other existing command.
