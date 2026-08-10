---
id: query-format-flags
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 2
status: done
depends-on: query-command-select-only
---

# `spekk query` Supports `--json`, `--tsv`, `--csv` Format Flags

## Description

`spekk query` reuses the same output formatter as `spekk list`. The default
is table; `--json`, `--tsv`, and `--csv` select machine-readable variants.

## Success Criteria

- `spekk query "SELECT id, status FROM assertions" --tsv` produces
  tab-separated output with a lowercase header.
- `spekk query "SELECT id, status FROM assertions" --json` produces a JSON
  array of objects where each key is the column name.
- `spekk query "SELECT id, status FROM assertions" --csv` produces RFC 4180
  CSV with a header row.
- `spekk query "SELECT id, status FROM assertions" --long` is accepted (no
  error) and behaves the same as default table since there is no intrinsic
  `file` column in arbitrary SQL results.
- Format flags do not affect the SQL execution itself — only the output
  rendering.
