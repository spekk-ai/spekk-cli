---
id: typo-report-in-validate
parent: branch-field-typo-guard
created: 2026-08-21T17:35:19Z
priority: 1
status: not_started
---

# `spekk validate` Reports a Branch Value That Is Almost a Real Branch

`spekk validate` reports an assertion whose `branch` names no ref but closely resembles one that exists, and it names the resembled ref. This catches the typo that silently strands an assertion outside the work queue.

## Success criteria

### Which values are reported
- The check covers assertions only, not parent specs. `internal/parser/parser.go:928` filters the work queue by an assertion's branch, so that is where a wrong value does harm.
- An assertion is a candidate when its status is neither `done` nor `draft`. A `done` assertion on a deleted branch is the normal end state of merged work.
- A value is reported when **both** hold:
  1. it does not name a branch in the ref set, and
  2. its Levenshtein distance to at least one branch in the ref set is 2 or less.
- A value that names no branch and resembles no branch is **not** reported. A merged and deleted branch is expected, and is not a problem.

### The ref set
- The ref set comes from `crossbranch.DiscoverAllBranches("")`, which returns the deduplicated union of local heads and remote-tracking refs by logical name, and keeps the current branch.
- When git is unavailable, or the directory is no git repository, or the call returns an error, the check is skipped in silence. `validate` must keep working over a fixture tree in a plain temporary directory.

### Output
- One line per **distinct** branch value, on stderr, with the number of assertions that carry it:

  ```
  Warning: branch "feat/retry-billing-webhok" names no branch (3 assertions). Did you mean "feat/retry-billing-webhook"?
  ```

- Grouping by distinct value is required, not cosmetic. The parser defaults an absent `branch` field to `"main"`, so on a repository whose trunk is `master` every assertion without the field carries `"main"`. Grouped, that is one line. Ungrouped, it is one line per assertion.
- The named ref is the one at the smallest distance. A tie is broken by the lexicographically first name, so the message is deterministic.
- Lines are sorted by the reported branch value, for stable and diffable output.
- The count uses singular or plural correctly: `(1 assertion)`, `(3 assertions)`.

### Exit code
- A typo report is a **warning, not a failure**. `validate` exits `0` when the only findings are typo reports, and its stdout still holds only the `validate: N specs, M assertions OK` line. Nearness is a heuristic, so it must never break CI.

**Note:** Levenshtein distance is the plain edit distance over the whole value, including the prefix and the `/`. The threshold is a fixed 2. Do not scale it with length, and do not add a flag for it.

**Tests:** `internal/validate/` — a tree whose assertion names a branch one edit from a real ref reports it and names the ref; a tree whose assertion names a branch far from every ref reports nothing; a `done` assertion on an absent branch reports nothing; three assertions sharing one bad value produce one line reading `(3 assertions)`; a tie between two equally near refs names the lexicographically first; the exit code stays `0` and stdout stays the summary line; a tree in a non-git directory reports nothing and does not error.
