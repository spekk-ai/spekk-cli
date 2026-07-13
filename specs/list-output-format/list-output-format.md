---
id: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
---

# `spekk list` — Human-Readable Table Output Format

## Overview

Change `spekk list` default output from JSON to a human-readable table, with
`--json` / `--tsv` / `--csv` flags for machine-readable variants. The table
format is modeled on `kubectl get` / `docker ps`: a header row followed by
space-padded columns.

The motivating reason is token efficiency for LLM consumers. At 503 assertions,
JSON key repetition and punctuation costs ~46K tokens. A table is ~18K tokens;
TSV is ~16K — a 2.5× reduction before any status filtering. Additionally,
tables communicate field semantics once in the header rather than repeating
key names per row.

## Design

### Default output: table

```
ID                            STATUS       PRI  TITLE
auth-permissions-roles        in_progress  1    Auth, Permissions & Roles
bom-save-performance          done         2    BOM Save Performance
```

- Header row in uppercase
- Columns: `ID`, `STATUS`, `PRI` (priority), `TITLE`
- Column widths derived from content (longest value in each column)
- Space-padded to align values; at least two spaces between columns
- No trailing whitespace on last column (TITLE)
- `--long` / `-l` adds a `FILE` column after `TITLE`

### `--tsv` flag

Tab-separated values. Same columns as table, no padding. The header row uses
the same names but lowercase and tab-separated. Reliable for shell pipelines
(`cut -f2`, `awk`, `sort -k3`).

```
id	status	pri	title
auth-permissions-roles	in_progress	1	Auth, Permissions & Roles
```

### `--json` flag

Current JSON output, unchanged. No regression. For `jq` pipelines.

### `--csv` flag

RFC 4180 CSV with a header row. Fields containing commas or quotes are quoted.

```
id,status,pri,title
auth-permissions-roles,in_progress,1,"Auth, Permissions & Roles"
```

### `--long` / `-l` flag

Adds the `file` / `FILE` column to any format (table, TSV, JSON, CSV). In
JSON mode, this is a no-op since the `file` field already appears in JSON output.

### Interaction with `--assertions-only`

When `--assertions-only` is set, the table output includes a `PARENT` column
after `PRI` and before `TITLE`:

```
ID                     STATUS  PRI  PARENT                 TITLE
some-assertion         draft   1    my-spec                Some Assertion Title
```

TSV and CSV include the `parent` field in the same column position.

### Interaction with `--status` and `--specs-dir`

Format flags are orthogonal to filter flags. Any combination of `--status`,
`--assertions-only`, `--specs-dir`, and a format flag is valid.

## Assertions

See `assertions/` for what must be true.

## Success Criteria

- Running `spekk list` with no format flag prints a table to stdout.
- Running `spekk list --json` prints the current JSON (no regression).
- Running `spekk list --tsv` prints tab-separated output with lowercase header.
- Running `spekk list --csv` prints RFC 4180 CSV with header row.
- Running `spekk list --long` (or `-l`) adds a file path column to any format.
- Column widths in table output are derived from content, not fixed.
- All format flags work with `--status`, `--assertions-only`, and `--specs-dir`.
