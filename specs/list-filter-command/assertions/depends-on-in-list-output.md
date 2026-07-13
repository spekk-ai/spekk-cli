---
id: depends-on-in-list-output
parent: list-filter-command
created: 2026-07-12T21:00:00Z
priority: 1
status: done
depends-on: assertions-only-flag
---

# `depends_on` Array Field Included in `spekk list` Output

## Description

`spekk list` and `spekk list --assertions-only` must include a `depends_on`
field on each assertion object. This enables reverse dependency queries without
grep: the agent calls `spekk list --assertions-only`, receives all assertions
with their `depends_on` arrays, and finds reverse edges by scanning the JSON.
Combined with `--status` filtering, this stays token-efficient.

The field is an array (not a string) for forward compatibility. If an assertion
has no `depends-on` frontmatter field, the array is empty. If it has one, the
array contains that single string.

## Success Criteria

- `HierarchyAssertion` struct in `internal/parser/output.go` gains a
  `DependsOn []string` field with JSON tag `"depends_on"`.
- `FormatHierarchy` populates `DependsOn` from each `Assertion.DependsOn`:
  empty string → `[]string{}`, non-empty string → `[]string{value}`.
- The flat assertion type used by `FormatAssertionsFlat` also gains a
  `DependsOn []string` field with JSON tag `"depends_on"` and is populated
  the same way.
- `spekk list --assertions-only` on a repo with assertions that have
  `depends-on` frontmatter shows non-empty `"depends_on"` arrays.
- `spekk list --assertions-only` on a repo with no `depends-on` frontmatter
  shows `"depends_on": []` (empty array, not null, not absent).
- `spekk next --all` output gains `depends_on` too, because `HierarchyAssertion`
  is shared with `FormatHierarchy`. This is intentional: the field is useful
  there as well and adding a new JSON key is backward-compatible for existing
  consumers that ignore unknown fields.
- Unit tests in `internal/parser/output_test.go` verify:
  - An assertion with `DependsOn = "foo"` produces `"depends_on": ["foo"]`.
  - An assertion with `DependsOn = ""` produces `"depends_on": []`.
