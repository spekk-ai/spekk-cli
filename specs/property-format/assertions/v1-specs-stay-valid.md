---
id: v1-specs-stay-valid
parent: property-format
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-property-format
depends-on: property-kind-is-read
---

# A v1 Spec Tree Is a Valid v2 Spec Tree

## Description

v2 makes properties stricter one at a time, where the value is high. A project must not have to change a v1 file to use the v2 binary.

## Success criteria

- For a spec tree that uses no `kind`, no `anchors`, no comment anchor and no evidence marker, the v2 `spekk validate` prints the same stdout and gives the same exit code as v1.50.0.
- For the same tree, every field that the v1 `spekk list --json` prints has the same value in the v2 output. v2 can add keys, but it does not remove or rename a key.
- The spec tree of this repository at `main` 299283b passes the v2 `spekk validate`. A test or a recorded run shows it.
- Each v1 assertion reads as a property of kind `example` with no anchors, no sections and no evidence.

**Tests:** `internal/index/property_format_test.go`.
