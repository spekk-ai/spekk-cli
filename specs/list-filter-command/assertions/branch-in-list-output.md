---
id: branch-in-list-output
parent: list-filter-command
created: 2026-08-21T00:00:00Z
priority: 1
status: done
depends-on: depends-on-in-list-output
---

# `branch` Field Included in `spekk list` and `spekk next --all` Output

## Description

`spekk list --json` and `spekk next --all` must include a `branch` field on each spec and assertion object. The parser already reads `branch` from the frontmatter and defaults it to `main`, and `spekk next` (single assertion) and `spekk --raw` already emit it. The two hierarchy and flat views dropped it, so an agent that enumerates work with `spekk list` could not tell which branch each assertion belongs to without a second call per record.

Adding a JSON key is backward-compatible: an existing consumer that ignores unknown fields is not affected.

The table, TSV, and CSV views of `spekk list` do not show a branch column. The JSON view is a superset of those views, not a mirror of them.

## Success Criteria

- `FlatAssertion` in `internal/parser/output.go` has a `Branch string` field with JSON tag `"branch"`, populated by `FormatAssertionsFlat` from `Assertion.Branch`.
- `HierarchyAssertion` has the same field, populated by `FormatHierarchy` from `Assertion.Branch`.
- `HierarchySpec` has the same field, populated by `FormatHierarchy` from `Spec.Branch`, so a spec and its assertions report the branch the same way.
- The tag has no `omitempty`, which agrees with `NextAssertionOutput`, `RawSpec`, and `RawAssertion`. An empty branch serializes as `""` and the key stays in the object.
- `spekk list --json` on a repo with branch-assigned assertions shows the assigned branch. An assertion with no `branch` frontmatter shows `"branch": "main"`, because the parser applies that default.
- Unit tests in `internal/parser/output_test.go` verify the field for both formatters, for an assigned branch and for an empty one.
