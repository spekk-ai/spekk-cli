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

The empty-result notice in table output states which filters found no matches. This distinguishes an empty specs directory from a filter with no matches.

## Success Criteria

- When `--status <value>` produces zero matching assertions, the table notice includes the status value (e.g., "No assertions match status 'draft'.").
- When `--priority <N>` produces zero matching assertions, the table notice includes the priority value.
- When both filters are active, the table notice includes both values.
- With no filters, the table notice states that no assertions were found in the specs directory.
- JSON uses the flat list object with an empty `assertions` array. TSV and CSV output contain their header only, as specified by `format-aware-empty`.
- Command tests check the notices and the machine-readable output.
