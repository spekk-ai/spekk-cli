---
id: query-shared-formatting
parent: sqlite-index
created: 2026-07-25T12:00:00Z
priority: 2
status: done
branch: feat/sqlite-index
depends-on: query-command-select-only
---

# Query Output Reuses the Shared Formatter

## Description

`spekk query` and `spekk list` must not carry two copies of table/TSV/CSV/JSON formatting. `internal/formatter` already renders these for `spekk list`; the query path reimplements them, and the copies have drifted — query's TSV does not sanitize control characters, and its JSON is hand-rolled with `%q` (not guaranteed-valid JSON, and it stringifies numbers). Both paths render from a shared, column-generic renderer so behavior is defined once and correct once.

## Success Criteria

- `internal/formatter` exposes column-generic renderers over `(headers []string, rows [][]string)` for table, TSV, CSV, and JSON.
- The existing `Row`-based `FormatTable`/`FormatTSV`/`FormatCSV` used by `spekk list` delegate to those renderers, and `spekk list`'s output is byte-for-byte unchanged (existing formatter tests still pass).
- The query path renders through the same renderers rather than its own copies; `csvQueryField` (a duplicate of the formatter's `csvField`) is removed.
- Query TSV sanitizes tab/CR/LF in cell values (no column corruption), matching `spekk list`.
- Query JSON is produced with `encoding/json` (valid escaping) while preserving SELECT column order.
