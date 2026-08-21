---
id: coach-requeues-with-not-started
parent: lock-is-a-live-claim
created: 2026-08-21T17:35:19Z
priority: 1
branch: fix/warning-discipline-and-lock-model
status: done
---

# The Coach Prompt Requeues an Edited Assertion With `not_started`

`specs/coach-agent/coach.prompt.md` tells the coach to write a status it can actually write. The current instruction names `in_progress`, which needs a lock only a builder session can mint.

## Success criteria

- In the "Status Rules (assertions only)" section of `specs/coach-agent/coach.prompt.md`, both rules change from `in_progress` to `not_started`:
  - `Updating assertion with status: done` becomes **Change to `status: not_started`**
  - `Updating assertion with status: failed` becomes **Change to `status: not_started`**
- The reason stays with the rule, in one short sentence: the assertion returns to the work queue so the builder re-implements it against the new requirements, and `not_started` needs no lock.
- The rule for an assertion already `in_progress` or `not_started` is unchanged: keep as is.
- The prompt states that the coach never writes a `locked-by` value. A lock names a live builder session, and the coach has no session to name.
- No other behavior in the prompt is removed or contradicted. This is an edit to three lines and one added sentence, not a rewrite of the section.
- `specs/builder-agent/builder.prompt.md` is checked for the same instruction, and corrected the same way if it holds one.

**Note:** This is prompt prose only. The validator half of the same problem is `unlocked-in-progress-is-legal`. Both are needed: the prompt change alone leaves a crashed builder's unlocked assertion failing validation, and the validator change alone leaves the coach writing a status whose meaning it cannot honor.

**Tests:** Prompt prose has no unit test. Verify by reading the rendered `spekk prompt coach` output and confirming that no rule tells the coach to write `in_progress`, and that the reproduction in GitHub issue #193 no longer has a wrong choice to make.
