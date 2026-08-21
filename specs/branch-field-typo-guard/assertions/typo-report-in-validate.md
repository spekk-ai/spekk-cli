---
id: typo-report-in-validate
parent: branch-field-typo-guard
created: 2026-08-21T17:35:19Z
priority: 1
branch: fix/warning-discipline-and-lock-model
status: done
---

# `spekk validate` Reports an Assertion Whose Branch Does Not Exist

`spekk validate` reports an assertion that is still in the work queue and names a branch that no ref matches. Such an assertion is invisible to `spekk next` for ever, whether the value is a typo or a branch somebody deleted.

## Success criteria

### Which values are reported
- The check covers assertions only, not parent specs. `internal/parser/parser.go:928` filters the work queue by an assertion's branch, so that is where a wrong value does harm.
- An assertion is a candidate when its status is neither `done` nor `draft`. A `done` assertion on a deleted branch is the normal end state of merged work, and a `draft` one is out of the queue by choice.
- A candidate is reported when its `branch` matches no name in the ref set. Plain set membership, and nothing more. Do not measure the distance to a near name, do not guess an intended name, and do not add a threshold.

### The ref set
- The ref set comes from `crossbranch.DiscoverAllBranches("")`, which returns the deduplicated union of local heads and remote-tracking refs by logical name, and keeps the current branch.
- When git is unavailable, or the directory is no git repository, or the call returns an error, the check is skipped in silence. `validate` must keep working over a fixture tree in a plain temporary directory.

### Output
- One line per **distinct** branch value, on stderr, with the number of assertions that carry it:

  ```
  Warning: branch "feat/retry-billing-webhok" matches no branch (3 assertions not done)
  ```

- Grouping by distinct value is required, not cosmetic. The parser defaults an absent `branch` field to `"main"`, so on a repository whose trunk is `master` every assertion without the field carries `"main"`. Grouped, that is one line. Ungrouped, it is one line per assertion.
- Lines are sorted by the reported branch value, for stable and diffable output.
- The count uses singular or plural correctly: `(1 assertion not done)`, `(3 assertions not done)`.

### Exit code
- The report is a **warning, not a failure**. `validate` exits `0` when the only findings are these reports, and its stdout still holds only the `validate: N specs, M assertions OK` line. A branch that is legitimately absent for a moment, such as one not yet pushed or fetched, must never break CI.

**Tests:** `internal/validate/branch_refs_test.go` — a queue-visible assertion naming an absent branch is reported; a `done` assertion naming the same absent branch is not; a `draft` one is not; an assertion naming a branch that exists is not; three assertions sharing one absent value produce one line reading `(3 assertions not done)`, and one assertion reads `(1 assertion not done)`; the exit code stays `0` and stdout stays the summary line; a tree in a non-git directory reports nothing and does not error.
