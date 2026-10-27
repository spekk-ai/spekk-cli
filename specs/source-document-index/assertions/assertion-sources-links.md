---
id: assertion-sources-links
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 2
status: done
branch: feature/source-document-index
depends-on: docs-status-and-freshness
---

# Specs and Assertions Cite Source Documents, and Validate Checks the Links

## Success criteria

- A spec or an assertion may have `sources: [id, ...]` in frontmatter. Each id names a document of kind `raw` or `doc`. The field is a custom field, so it is in `frontmatter_fields` (key `sources`) with no schema change.
- `spekk validate` prints a warning, and never a failure, for: a `sources` id that names no document (`<file>: sources: unknown document "<id>"`); a cited document whose state is `missing_raw`; and, for the corpus, each `stale`, `no_summary`, `missing_raw`, and `no_stub` document (a `pointer` is not a problem) (one warning line each, with the command to run). The exit code does not change.
- `spekk validate` reads the corpus from the files (the same scan the index uses), not from `index.db`.
- The observer prompt says: when an assertion has `sources`, read the cited documents with `spekk docs show` before you report that the spec is underspecified, and report a value that the source holds but the spec does not.

**Tests:** `internal/validate/validate_test.go`: one validate test: an unknown id gives the warning and exit 0, a known id gives no warning.
