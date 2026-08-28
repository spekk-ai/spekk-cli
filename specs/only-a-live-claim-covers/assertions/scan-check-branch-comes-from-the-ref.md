---
id: scan-check-branch-comes-from-the-ref
parent: only-a-live-claim-covers
created: 2026-08-28T12:00:00Z
priority: 1
branch: fix/only-a-live-claim-covers
status: done
---

# The `branch` of a Covered Result Names the Ref the Observation Came From

A `covered` result tells a person where to go and look. The field must be a fact read from git, not a name rebuilt from a slug.

## Success criteria

- `internal/observation` exports a function that reduces a fully-qualified ref to its logical branch name: `refs/heads/observer/x` and `refs/remotes/origin/observer/x` both give `observer/x`. `isMainRef` uses the same reduction, so one function holds the rule.
- The `covered` result of `spekk observer scan-check` sets `branch` from `covering.Ref` through that function. `BranchName(covering.Slug)` is no longer used there.
- The `clear` result is unchanged: it still reports `BranchName(resolved)`, because that branch does not exist yet and the slug is all there is.

**Note:** with `covering-needs-the-owning-branch` in place the two values agree by construction, because only the owning branch can cover. The field is still read from the ref, so it stays a fact if the covering rule ever widens again.

**Tests:** `cmd/spekk/observer_test.go` — a `covered` result found at a remote-tracking ref reports `branch` as the logical branch name and `ref` as the fully-qualified ref.
