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

`spekk list --status <value>` returns a filtered hierarchy containing only
assertions whose `status` field matches the given value.

## Success Criteria

- `spekk list --status draft` returns a JSON hierarchy where every assertion in
  every `"assertions"` array has `"status": "draft"`.
- Assertions with a different status are absent from the output.
- Specs that have at least one matching assertion are included in `"specs"`.
- Specs that have zero matching assertions are excluded from `"specs"`.
- The rolled-up spec `status` field is NOT used for filtering — a spec is
  included based solely on whether it has matching assertions.
- `spekk list --status done` includes only done assertions and their parent specs.
- `spekk list --status not_started` includes only not_started assertions.
- `spekk list --status in_progress` includes only in_progress assertions.
- `spekk list --status failed` includes only failed assertions.
- An invalid status value (e.g. `--status bogus`) causes the command to exit
  non-zero and print a message listing the valid status values.
- Filtering does not affect `spekk next --all` or any other existing command.
