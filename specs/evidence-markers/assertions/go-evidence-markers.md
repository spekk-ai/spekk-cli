---
id: go-evidence-markers
parent: evidence-markers
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-evidence-markers
---

# Spekk Finds Each Go Evidence Marker and the Test It Marks

## Description

A Go marker is a `// spekk: <id>` comment, with an optional ` evidence=<level>` after the ID and nothing more, in the doc comment block directly above a `func TestXxx(t *testing.T)` or `func FuzzXxx(f *testing.F)` in a `_test.go` file.

## Success criteria

- After `spekk index`, the `evidence` table has one row `(assertion_id, level, language, file, line, test)` for each Go marker: `language = 'go'`, `test` is the function name, and `file` is the project-relative path. The package directory of `file` and `test` together name the test.
- A function with two or more marker comments has one row for each marker. One test can check more than one property.
- `evidence=property` gives level `property`. A marker with no `evidence=` has level `example`.
- A `spekk:` comment that is separated from the function by a blank line or by code is not a marker. It is an anchor and follows `comment-anchors`.
- A marker above a function that is not a `Test` or `Fuzz` function is not a marker. It is an anchor.
- A marker comment is never also an anchor: it adds no row to the `anchors` table.
- **Tests:** `internal/anchors/scan_test.go` and `internal/index/evidence_test.go` check levels, multiple markers, function position, anchor separation, index rows, and HEAD refresh.
