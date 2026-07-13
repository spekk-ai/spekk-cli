---
id: limit-flag
parent: list-output-format
created: 2026-07-13T01:00:00Z
priority: 3
status: not_started
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# `--limit N` Caps Output to First N Assertions

## Description

`spekk list --limit N` outputs at most N assertions. Applied after filtering
and sorting so the user gets the top-N by whatever sort/filter criteria they
chose. Works with all output format flags.

## Success Criteria

- `spekk list --limit 5` outputs at most 5 assertions (fewer if fewer match).
- When combined with `--sort-by priority`, `--limit 5` outputs the 5
  highest-priority assertions (priority 1 first).
- `--limit 0` outputs an empty list (no error).
- A negative N causes a non-zero exit with a descriptive error message.
- A non-integer N causes a non-zero exit with a descriptive error message.
- `--limit` applies after `--status` and `--priority` filters: `spekk list
  --status not_started --limit 3` returns the first 3 not-started assertions.
- The total count of matching assertions (before the limit) is NOT printed
  to stdout (only the limited rows are shown; the caller can count).
- `--limit` works with `--assertions-only`, `--json`, `--tsv`, `--csv`, `--long`.
