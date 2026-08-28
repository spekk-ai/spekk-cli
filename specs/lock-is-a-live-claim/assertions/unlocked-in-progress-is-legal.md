---
id: unlocked-in-progress-is-legal
parent: lock-is-a-live-claim
created: 2026-08-21T17:35:19Z
priority: 1
branch: fix/warning-discipline-and-lock-model
status: done
---

# `validate` Accepts an `in_progress` Assertion That Carries No Lock

`checkLockState` stops requiring a `locked-by` value on an `in_progress` assertion. The lock records a live claim by a builder session, so its absence means nobody holds the assertion, which is a legal state.

## Success criteria

- In `internal/validate/validate.go`, the `case "in_progress":` branch of `checkLockState` no longer adds the failure `status is in_progress but locked-by is missing`. That failure message is deleted.
- The `default:` branch is unchanged. A `done`, `failed`, `not_started`, or `draft` assertion that carries a non-empty `locked-by` is still a failure, with its existing message: `status is %s but locked-by is set (%q); only in_progress may carry a lock`.
- The doc comment on `checkLockState` is rewritten to state the rule that now holds: only `in_progress` may carry a lock, and it need not carry one. It must not describe the removed requirement.
- The lock rule reads the same in all three places that state it. Update:
  - `specs/spec-validation/spec-validation.md`, invariant 5 in "Invariants enforced"
  - `specs/spec-validation/assertions/validate-command.md`, invariant 5 under "Invariants checked" and the matching line in its "Tests" list
  - `docs/cli-reference.md` and `docs/concepts.md` wherever they state the pairing
- `spekk validate` passes on the reproduction in GitHub issue #193: a tree whose only change is `status: done` edited to `status: in_progress`, with no `locked-by` added, exits `0`.

**Note:** This is a deliberate narrowing of an invariant, so an existing test asserts the old behavior. Change that test to assert the new rule rather than deleting it, because the unlocked `in_progress` case still needs coverage — with the opposite expectation.

**Tests:** `internal/validate/validate_test.go` and `cmd/spekk/validate_test.go` — an `in_progress` assertion with no `locked-by` passes (this inverts the existing case); an `in_progress` assertion with a `locked-by` still passes; a `done` assertion carrying a `locked-by` still fails; the same for `failed`, `not_started`, and `draft`.
