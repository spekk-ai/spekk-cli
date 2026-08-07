---
id: machine-readable-cross-branch
parent: show-cross-branch
created: 2026-08-07T23:00:00Z
priority: 2
status: done
branch: feat/list-cross-branch
depends-on: classify-cross-branch-state
---

# Cross-Branch State Is Available as Machine-Readable CLI Output

## Description

The HTML explorer is the only consumer of the cross-branch classification an
agent cannot use. `spekk list --cross-branch` exposes the same `Classify`
result — one row per changed (file, branch) pair — as data, so an observer
agent can reason about spec drift on feature branches before merge (for
example, an assertion `done` on `main` that a feature branch moves back to
`in_progress`).

## Success Criteria

- `spekk list --cross-branch` prints one row per `Classify` contribution
  with columns `path`, `branch`, `state` (`incoming_add` | `incoming_mod` |
  `conflict` | `incoming_del`), `degraded`, `old_status`, `new_status`.
- `--json` emits an array of objects with those keys; `degraded` is a JSON
  boolean, empty statuses are omitted, and an empty result set renders `[]`
  (never `null`). `--tsv` and `--csv` emit the same columns through the
  shared formatter renderers.
- `--branch-filter <glob>` restricts the comparison branches, same as in
  `spekk show --cross-branch`.
- The foreign-file `Meta` payload (full parsed content for the explorer)
  never appears in the machine output.
- `--branch-filter` without `--cross-branch`, and `--status` combined with
  `--cross-branch`, are clear errors.
- The read-only guarantee of the cross-branch engine applies unchanged: the
  command never touches the working tree, index, or refs.
