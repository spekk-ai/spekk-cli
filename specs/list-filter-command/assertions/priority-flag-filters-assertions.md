---
id: priority-flag-filters-assertions
parent: list-filter-command
created: 2026-07-13T01:00:00Z
priority: 2
status: not_started
branch: feat/list-filter-by-status
depends-on: status-flag-filters-assertions
---

# `--priority` Flag Filters Assertions by Priority Level

## Description

`spekk list --priority <N>` returns a filtered hierarchy containing only
assertions whose `priority` field equals the given integer value. Supports
combining with `--status` and `--assertions-only`.

## Success Criteria

- `spekk list --priority 1` returns only assertions with `priority: 1`.
- `spekk list --priority 2` returns only assertions with `priority: 2`.
- An assertion with any other priority value is absent from the output.
- `--priority` and `--status` can be combined: `spekk list --status done --priority 1`
  returns only assertions that are both done AND priority 1.
- `--priority` works with `--assertions-only` (flat list mode).
- An invalid value (non-integer or negative) causes a non-zero exit with a
  descriptive error message.
- `--priority 0` returns an empty list (there are no priority-0 assertions
  by convention, but the command should not error).
- Filtering does not affect `spekk next`, `spekk next --all`, or any
  other existing command.
