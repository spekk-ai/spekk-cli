---
id: filtered-empty-message
parent: list-filter-command
created: 2026-07-13T00:00:00Z
priority: 1
status: done
depends-on: status-flag-filters-assertions
---

# Contextual Message When Filter Produces No Results

## Description

`parser.FormatEmpty()` always says "No specifications found in specs/ directory"
even when specs exist but the `--status` filter matched nothing. A caller running
`spekk list --status draft` and getting that message would think the specs
directory is missing, when the real reason is there are no draft assertions.

The message must include the filter value so callers can tell the difference.

## Success Criteria

- When `--status <value>` is active and the filter produces zero matching
  assertions, the empty-result message mentions the filter value (e.g.,
  "No assertions match status 'draft'.").
- The contextual message is emitted in the correct format (JSON for default/JSON,
  header-only for TSV/CSV) — this combines with the format-aware-empty assertion.
- When no `--status` filter is active, the existing "No specifications found"
  message is unchanged.
- A new `FormatEmptyFiltered(status string)` (or equivalent) is added to the
  `internal/parser` package and returns a JSON object with the contextual message.
- A unit test in `internal/parser/output_test.go` verifies the returned message
  contains the status value.
