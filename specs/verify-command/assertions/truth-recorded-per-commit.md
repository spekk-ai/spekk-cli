---
id: truth-recorded-per-commit
parent: verify-command
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: truth-at-head-follows-the-evidence
---

# Each Verify Run Records the Truth of Every Property for Its Commit

## Description

`spekk verify` records the commit, the results and the truth of each property in `.spekk/index.db`, so that `spekk query` and the display commands read them, and so that a later commit does not hide what was true at an earlier one.

## Success criteria

- The index has three tables:
  - `verify_runs(id, commit, dirty, started, finished)`: one row for each run. `commit` is the full HEAD SHA. `dirty` is 1 when the work tree had uncommitted changes.
  - `verify_results(run_id, assertion_id, file, test, level, outcome)`: one row for each mapped result. `outcome` is `passed`, `failed`, `error` or `skipped`.
  - `verify_truth(run_id, assertion_id, truth, level)`: one row for each property in the spec tree, also for properties with no checks.
- The rows of runs for earlier commits stay after a new run.
- A content rebuild of the index (a spec edit, or a new HEAD) keeps all three tables and their rows.
- A force rebuild or a schema-version rebuild drops them, as `force-flag-rebuilds-from-scratch` says. After it, each property is `unchecked` until the next run.
- Stdout of `spekk verify` ends with one summary line: `verify: <n> verified, <n> failing, <n> unchecked, <n> unconfirmed`. Each failing property has one line before it, with its ID and the names of its failed tests.
- `spekk query "SELECT assertion_id, truth FROM verify_truth WHERE run_id = (SELECT MAX(id) FROM verify_runs)"` returns one row for each property.
