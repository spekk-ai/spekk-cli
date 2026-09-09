---
id: review
description: Review what was just built against the assertions marked done. Six lenses, each with a remedy. Fixes what it finds, and gates the push.
created: 2026-09-03T19:40:00Z
priority: 1
---

# Review

Reviews what was just built against the assertions that were marked `done`. It is the verify phase of the dev loop, written as a procedure. It runs in the session that built the code, where it reuses the context that the build made, or in a fresh session when independence matters more than context.

**The review fixes what it finds.** The builder role may write code, so a defect found here is fixed here. This is the difference from an observer skill, which may only recommend. The review writes no observation file.

**The push waits for the review.** Nothing built in this session is pushed until the review reports.

**CLI Command:** `spekk builder review` (fresh session) or `spekk skill show builder review` (adopt in the current session)

## Triggers

- "review what we built"
- "verify the feature"
- "run the review"
- Phase 3 of the `spekk-dev-loop` skill, after the queue for a feature is empty
- Before a push, when the session marked one or more assertions `done`

## Workflow

### Scope

The scope is the assertions marked `done` on the current branch since it left its base branch, plus the diff from that base to `HEAD`. Find both from git:

```bash
BASE=$(git merge-base main HEAD)                       # use the project's base branch when it is not main
git diff --name-only "$BASE"..HEAD -- 'specs/*/assertions/*.md'   # the assertions touched on this branch
git diff "$BASE"..HEAD                                  # the code under review
```

Read each touched assertion file and keep the ones whose status is now `done`. When the session runs on the base branch itself, the diff is empty, so the user names the commit range instead.

Read the code as it is. Do not review from the summary in memory.

### Lenses

Apply the six lenses in this order. Each one names its remedy.

**1. Every criterion, against the real code.** Re-read every success criterion of every in-scope assertion, and check it against the code as it is. For a criterion with non-obvious behavior (an output format, an ordering rule, an edge case, a library default), trace one concrete input through the code line by line and confirm the result. A criterion that is not met is fixed now. A criterion that cannot be fixed now sets the assertion to `failed`, which is the status the model already has for a confirmed gap. Do not leave an unmet criterion under `done`.

**2. Every test earns its place.** A test is kept only when it would fail if the behavior it pins were broken. Delete a test that passes when that behavior is broken, a test that restates the implementation instead of pinning behavior, a test that duplicates another test, and a test that exercises a mock instead of the real code path. The bar is lean and high value. Fewer, stronger tests beat many weak ones.

**3. Nothing beyond what the assertions ask for.** Remove generality, configuration, and abstraction that no assertion requires: a flag nobody reads, an interface with one implementation, a helper with one caller that adds a level of indirection, an option kept for a future that no spec describes. Walk the diff hunk by hunk. A hunk that no assertion accounts for is reverted, or the reason it stays is stated in the report.

**4. Errors are loud.** An error that is dropped, replaced with a default, or caught by a broad handler is a defect. Fix it so the failure reaches the caller or the log with its cause.

**5. The spec tree is sound.** `spekk validate` exits 0. `spekk next` succeeds. No assertion carries a `locked-by` field unless its status is `in_progress`. Each `**Tests:**` link in an in-scope assertion points to a file that exists.

**6. The diff is fit to publish.** Nothing in the diff that the project's own rules forbid: no secret, no credential, no private name, no reference to a repository other than this one. Assume the repository is public unless the project says otherwise.

### Close

Run the checks in the Validation section, then write the report described in the Output Format section. Commit the fixes. Only then push.

## Output Format

The output is the fixes in the working tree, plus a short report in the session. No file under `observations/` is written. No new frontmatter field is added to an assertion.

The report has four parts:

```
Review of <branch> against <base>

Assertions
  <assertion-id>   ok        every criterion met
  <assertion-id>   fixed     criterion 3 was not met; <what changed>
  <assertion-id>   failed    criterion 2 needs <decision>; set to failed

Fixed
  - <file>: <what was wrong, what changed>

Deleted
  - <test or code>: <which lens, why>

Open
  - <what stays open, and who decides>
```

An empty part reads `none`.

## Validation

Before the report:

- `spekk validate` exits 0 and `spekk next` succeeds
- The full test suite passes after the fixes and the deletions
- Every assertion in scope is `done` or `failed`; none is `in_progress` with a stale lock
- Every diff hunk is accounted for by an assertion or by a stated reason in the report
- No file under `observations/` was written
- The push has not happened yet

## Examples

### Example 1: In-session review at the end of a dev loop

```
$ spekk skill show builder review
> (the session adopts the skill and reviews its own branch)

Review of feature/list-limit against main

Assertions
  limit-flag             ok
  limit-flag-validation  fixed     "--limit 0" returned every row; now an error

Fixed
  - internal/list/flags.go: a zero limit fell through to "no limit"; it now fails with a message

Deleted
  - internal/list/flags_test.go TestParseLimit_Positive: duplicated TestParseLimit table case

Open
  none
```

### Example 2: Fresh-context review of a large change

```
$ spekk builder review
> Starting Builder Agent with skill: review

Review of feature/index-rebuild against main

Assertions
  index-rebuild-on-change  ok
  index-survives-restart   failed    criterion 2 (a partial write is rolled back) has no test and the code does not roll back; needs a decision on the journal format

Fixed
  none

Deleted
  none

Open
  - index-survives-restart: journal format; set to failed for the coach
```
