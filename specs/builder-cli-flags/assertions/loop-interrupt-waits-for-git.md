---
id: loop-interrupt-waits-for-git
parent: builder-cli-flags
created: 2026-11-27T23:00:00Z
priority: 1
status: done
branch: fix/loop-builder-waits-for-commit-step
---

# The loop builder exits only after its git and spekk next children

When Ctrl+C or SIGTERM makes `spekk loop builder` exit and no builder agent is active, the loop does not leave a child process that writes to the repository, and it does not start a builder agent after the signal.

This guarantee applies when the signal reaches only the loop process (for example, `kill -INT <pid>` or `kill -TERM <pid>`). A Ctrl+C in a terminal also signals the git or `spekk next` child in the same process group, and that child can stop before it finishes.

## Success Criteria

- If the signal arrives while the commit step runs (`git status`, `git add`, `git commit`), the loop waits for the step to finish, then prints the completion-count line and exits with status 0. The commit that the step makes is in the repository.
- If the signal arrives while `spekk next` runs, the loop waits for it, then prints the completion-count line and exits with status 0. No builder agent starts after the signal.
- A second SIGINT or SIGTERM that arrives while the loop waits for a child exits the loop at once, with status 128 plus the signal number (130 for SIGINT, 143 for SIGTERM).
- A signal that arrives while the loop sleeps or waits for work exits at once, as before.
- During an active build, Ctrl+C still goes to the builder agent, as `exit-when-done-flag` and `builder-start-interrupt` say.

**Tests:** cmd/spekk/builder_loop_exit_test.go (TestLoopBuilderInterruptWaitsForCommitStep, TestLoopBuilderSignalDuringNextStartsNoAgent)
