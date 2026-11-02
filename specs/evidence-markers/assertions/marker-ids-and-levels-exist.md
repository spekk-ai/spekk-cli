---
id: marker-ids-and-levels-exist
parent: evidence-markers
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: go-evidence-markers
---

# Every Evidence Marker Names a Property and a Level That Exist

## Description

A marker that names no property gives evidence for nothing, and an unknown level cannot take a place on the ladder. `spekk validate` rejects both, as it rejects an unknown anchor ID.

## Success criteria

- For a Go or Python marker whose ID is not an assertion ID, `spekk validate` prints one failure line `<path>:<line>: evidence marker names unknown property <id>` and exits 1.
- For a marker whose level is not one of `example`, `property`, `contract`, `bounded` or `proof`, `spekk validate` prints one failure line `<path>:<line>: unknown evidence level <level>` and exits 1.
- Markers in the files that `comment-anchors` skips (`.md`, `testdata`, binary, ignored) are not read.
- A tree whose markers are all valid validates as before: the same summary line and exit 0.
