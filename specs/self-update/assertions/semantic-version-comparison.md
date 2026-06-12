---
id: semantic-version-comparison
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 1
status: done
---

# Versions Are Compared Semantically, Not Lexically

## Description

The update decision uses numeric, component-wise version comparison
(`ParseVersion` / `IsNewer` in `internal/update/update.go`), so `1.10.0` is
correctly newer than `1.9.9` and an update only happens when the remote
version is strictly newer.

## Success Criteria

- `ParseVersion` accepts versions with or without a leading `v`
  (`1.2.3`, `v1.2.3`) and splits them into numeric components
- Anything after the first `-` is ignored (`1.2.3-dirty` → `1.2.3`)
- Non-numeric input (e.g. `dev`) parses as `[0]` rather than panicking
- `IsNewer(a, b)` is true only when `a` is strictly newer: component-wise
  numeric comparison, missing components treated as `0`
  (`1.1.0 > 1.0.0`, `2.0.0 > 1.9.9`, `1.0.0` vs `1.0.0` → false)
- When the latest release is not newer than the running version,
  `spekk update` prints "Already on latest version (<version>)" and exits
  successfully without downloading anything

**Tests:** `TestParseVersion`, `TestIsNewer` in
`internal/update/update_test.go`.
