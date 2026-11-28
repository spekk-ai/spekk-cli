---
slug: loop-builder-sigterm-during-build-unspecified
type: underspecified_spec
severity: medium
status: resolved
created: 2026-11-28T12:00:00Z
announced: 2026-11-28T20:08:44Z
affected:
  - internal/agent/loop.go
  - internal/agent/builder.go
  - specs/builder-cli-flags/assertions/loop-interrupt-waits-for-git.md
  - specs/builder-cli-flags/assertions/exit-when-done-flag.md
---

# No Spec Says What `spekk loop builder` Does on SIGTERM During an Active Build

## Issue Description

The `builder-cli-flags` assertions say what happens to `spekk loop builder` on Ctrl+C during a build, and on SIGINT or SIGTERM when no builder agent is active. No assertion covers SIGTERM while a builder agent runs. `loop-interrupt-waits-for-git` (added in #380) leaves that case out on purpose: its guarantee holds only "when ... no builder agent is active".

Two implementations both meet every current criterion, and they behave very differently for the user:

1. **Exit at once (the current code).** The loop prints "Exiting gracefully...", prints the completion count, and exits with status 0. The `claude` builder agent, started with `--dangerously-skip-permissions`, keeps running as an orphan. It goes on editing the repository after the loop says it has exited, and no commit step ever runs for its work.
2. **Forward and wait.** The loop sends SIGTERM to the agent, waits for it to exit, and only then exits. No process writes to the repository after the loop is gone.

#380 removed exactly this kind of orphan writer for the `git` and `spekk next` children (an orphan `git` wrote `.git/index` during test cleanup). A builder agent left behind by SIGTERM is the same hazard, and it lasts longer. SIGTERM is the signal that `kill <pid>`, process supervisors, and stop scripts send.

## Evidence

- `internal/agent/loop.go:108`: the handler forwards a signal to the agent only when `sig == syscall.SIGINT && active.isSet()`. For SIGTERM it falls through to `exiting = true` (line 114) and `go exitAfterSteps()` (line 117).
- `internal/agent/loop.go` `exitAfterSteps`: it locks `steps`, which is held only around `spekk next` (line 138) and the commit step (line 218), never around `launchClaude` (line 199). So during a build it gets the lock at once and calls `os.Exit(0)` while the agent is still running.
- `internal/agent/builder.go` `launchClaude`: the child gets no `SysProcAttr` (no `Pdeathsig`, no process group handling), so nothing stops it when the loop exits.
- `specs/builder-cli-flags/assertions/loop-interrupt-waits-for-git.md`: "When Ctrl+C or SIGTERM makes `spekk loop builder` exit and no builder agent is active ..." The examples it gives are `kill -INT <pid>` / `kill -TERM <pid>`, which reach only the loop process.
- `specs/builder-cli-flags/assertions/exit-when-done-flag.md`: "During an active build, Ctrl+C reaches the child, and the parent waits for the child." It says nothing about SIGTERM.

## Impact

An operator who stops a builder loop with `kill`, or with a supervisor that signals only the main PID, gets status 0 and a "graceful" message, while an unattended agent with skipped permissions keeps changing the working tree. A later loop or a person can then start work on a tree that is still being written to. Because the spec is silent, a builder has no basis for choosing the safe behavior, and a test cannot pin it.

## Recommendation

Add an assertion that states the intended result. The remedy commit on this branch proposes one, `loop-sigterm-during-build`: SIGTERM during an active build goes to the agent, and the loop waits for the agent before it exits. If the current exit-at-once behavior is what you want instead, change the assertion to say so, and say that the agent outlives the loop.
