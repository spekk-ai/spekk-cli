---
id: loop-sigterm-during-build
parent: builder-cli-flags
created: 2026-11-28T20:07:41Z
priority: 1
status: done
---

# The loop builder does not leave a builder agent running when SIGTERM stops it

When `spekk loop builder` receives SIGTERM while a builder agent runs, the loop does not exit until that agent exits. No builder agent keeps running after the loop has exited.

## Success Criteria

- When SIGTERM reaches only the loop process (for example, `kill -TERM <pid>`) during an active build, the loop forwards SIGTERM to the builder agent and waits for the agent to exit.
- After the agent exits, the loop does not run the commit step and does not start another builder agent. It prints the completion-count line and exits with status 0. The count does not include the interrupted build.
- A second SIGINT or SIGTERM while the loop waits for the agent exits the loop at once with status 128 plus the signal number, as `loop-interrupt-waits-for-git` says.
- Ctrl+C (SIGINT) during an active build behaves as before: it goes to the agent, and the loop keeps running after the agent exits (`exit-when-done-flag`).

**Tests:** cmd/spekk/builder_loop_exit_test.go (TestLoopBuilderSigtermDuringBuildWaitsForAgent)
