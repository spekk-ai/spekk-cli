---
id: property-kind-is-read
parent: property-format
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-property-format
---

# Each Property Has Exactly One Kind, and an Absent Kind Is `example`

## Description

The parser reads the `kind` key of an assertion's front matter. The value is one of `invariant`, `ordering`, `response`, `authorization` or `example`. A v1 assertion has no `kind`, and it reads as `example`. Later specs use the kind for strictness (`strictness-config`).

## Success criteria

- `kind: invariant` (and each of the other four values) gives that kind on the parsed assertion and in a new `kind` column of the `assertions` table of `.spekk/index.db`.
- An assertion with no `kind` key has kind `example` in the parsed model and in the index. The `kind` column is never empty or NULL.
- An unknown value (for example `kind: invarient`) makes `spekk validate` print one warning to stderr that names the file and the value. The exit code does not change, and the assertion has kind `example`.
- A comment after the value is not part of it (`kind: invariant  # note`), as `frontmatter-grammar` says.
- The `frontmatter_fields` table keeps its rows for `kind`, as for any custom field in v1.

**Tests:** `internal/parser/property_format_test.go`, `cmd/spekk/property_format_test.go`, `internal/index/property_format_test.go`.
