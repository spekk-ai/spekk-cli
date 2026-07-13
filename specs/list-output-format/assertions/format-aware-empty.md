---
id: format-aware-empty
parent: list-output-format
created: 2026-07-13T00:00:00Z
priority: 1
status: done
branch: feat/spekk-list
---

# Empty Results Respect the Active Format Flag

## Description

When `spekk list` finds no matching assertions (either because the specs
directory is empty or because the `--status` filter matched nothing), it must
emit output in the format the caller requested — not always a JSON blob.

Giving a TSV consumer a JSON object (even a small one) is a contract violation.

## Success Criteria

- `spekk list --tsv` with no matching assertions outputs a tab-separated header
  row only (no data rows, no JSON), then exits 0.
- `spekk list --csv` with no matching assertions outputs a CSV header row only
  (CRLF terminated per RFC 4180), then exits 0.
- `spekk list` (table, default) with no matching assertions outputs a meaningful
  message (the existing `FormatEmpty` JSON blob or a plain text notice), exits 0.
- `spekk list --json` with no matching assertions outputs the empty JSON message
  (FormatEmpty or equivalent), exits 0.
- Both the "no specs directory" path and the "filter produced empty result"
  path are covered by this format-aware behavior.
- A unit test (in `cmd/spekk/list_test.go` or similar) exercises `--tsv` and
  `--csv` with an empty fixture and asserts the output starts with the expected
  header row (not `{"status"`).
