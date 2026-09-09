---
id: priority-flag-filters-assertions
parent: list-filter-command
created: 2026-07-13T01:00:00Z
priority: 2
status: done
depends-on: status-flag-filters-assertions
---

# `--priority` Flag Filters Assertions by Priority Level

## Description

`spekk list --priority <N>` returns only assertions whose `priority` field equals the given integer. It uses the current flat list output and works with the existing status filter and output formats.

## Success Criteria

- `spekk list --priority 1` returns only assertions with `priority: 1`.
- `spekk list --priority 2` returns only assertions with `priority: 2`.
- An assertion with any other priority value is absent from the output.
- `--priority` and `--status` can be combined: `spekk list --status done --priority 1` returns only assertions that are both done and priority 1.
- `--priority` works with the table, JSON, TSV, and CSV output formats, with `--long`, and with the accepted compatibility flag `--assertions-only`.
- A missing, non-integer, negative, or overflowing value causes a nonzero exit with an error that identifies `--priority`.
- Values must follow a space. The unsupported `--priority=1` form fails with a nonzero exit instead of returning an unfiltered list.
- The command accepts one `--priority` flag. A repeated flag fails instead of hiding a missing or invalid value.
- A nonnegative value outside the stored priority range, including `0`, succeeds with no matching assertions. JSON contains an empty `assertions` array, TSV and CSV contain their header, and the table output reports that no assertions match the filters.
- `--priority` with `--cross-branch` fails with an error because cross-branch rows describe file changes.
- Help and the CLI reference describe the flag and its combination with `--status`.
- Filtering does not affect `spekk next`, `spekk next --all`, or other commands. Omitting `--priority` includes all priority levels.

## Verification

`cmd/spekk/list_test.go` checks priority selection, combined filters, all output formats, empty results, and invalid arguments. The parser tests check the shared filtering behavior.
