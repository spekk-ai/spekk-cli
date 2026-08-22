---
id: no-convention-warning
parent: branch-field-typo-guard
created: 2026-08-21T17:35:19Z
priority: 1
branch: fix/warning-discipline-and-lock-model
status: done
---

# The Branch Field Accepts Any Name Git Accepts

`validateBranch` stops judging a branch value against a naming convention. A value that git itself accepts produces no warning, whatever its shape.

## Success criteria

- `standardBranchNames`, `standardBranchTypes`, and `standardBranchPattern` are deleted from `internal/parser/parser.go`, together with the `Warning: Field 'branch' uses non-standard pattern` message and its `fmt.Fprintf(os.Stderr, ...)` call.
- `validateBranch` keeps every hard error it has today, unchanged, because each rejects a name git itself refuses:
  - a leading or trailing `/`
  - a character outside `validBranchPattern`
  - `..` anywhere, a trailing `.`, or a trailing `.lock`
- `validateBranch` writes nothing to stderr. After this change the function returns an error or nil, and produces no output.
- These values now pass with no warning: `dana/apx-12-retry-billing-webhook`, `sam/PROJ-441`, `temporary-target`, `release/1.22.0`, `myfeat/thing`.
- These values still fail with an error: `/leading`, `trailing/`, `feat/th..ing`, `feat/thing.`, `feat/thing.lock`, `feat/thing space`.
- `internal/parser/branch_pattern_test.go` is reduced to the hard-error cases. Every test that asserts the convention warning is deleted rather than inverted, because the behavior it pinned is gone.

**Note:** Deleting the warning is the whole of the fix for issue #192's volume problem. `spekk next`, `spekk list`, `spekk status`, and `spekk show` print zero branch warnings afterwards, so no summary line, no grouping, and no `--quiet` flag is needed for them.

**Tests:** `internal/parser/branch_pattern_test.go` — each hard-error value above returns an error; each accepted value above returns nil; a test captures stderr across a `validateBranch` call over the accepted values and asserts it is empty.
