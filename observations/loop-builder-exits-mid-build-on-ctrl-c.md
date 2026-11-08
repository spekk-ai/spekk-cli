---
slug: loop-builder-exits-mid-build-on-ctrl-c
type: code_spec_misalignment
severity: medium
status: open
created: 2026-10-28T12:00:00Z
announced: 2026-10-28T19:02:38Z
affected:
  - internal/agent/loop.go
  - specs/builder-cli-flags/assertions/exit-when-done-flag.md
  - cmd/spekk/builder_loop_exit_test.go
---

# `spekk loop builder` Exits Mid-Build on Ctrl+C Instead of Waiting for the Child

## Issue Description

The `exit-when-done-flag` assertion is marked `done`. It states that during an
active build, Ctrl+C reaches the child, the parent waits for the child, and a
continuous builder loop keeps running after the child exits. Its other criteria
cover both continuous commands, `spekk builder` and `spekk loop builder`.

`spekk builder` meets this criterion: `RunBuilder` forwards SIGINT to the active
child through `processHolder` and keeps looping. `spekk loop builder` does not.
`RunBuilderLoop` registers one signal handler. Whenever SIGINT or SIGTERM
arrives, it prints the completion count and calls `os.Exit(0)` straight away,
whether or not a build is running. It does not forward the signal, does not wait
for the child, and does not keep the loop running. It also skips the
`git add . && git commit` step for the interrupted session.

## Evidence

- `internal/agent/loop.go:84-89`: the handler goroutine runs `sig := <-sigCh`,
  logs, prints `formatLoopComplete(...)`, and then calls `os.Exit(0)`. It does
  not check whether a child is active.
- `internal/agent/loop.go:152`: `launchClaude(..., nil)` passes no
  `processHolder`, so nothing can forward the signal to the child or wait for it.
- `internal/agent/builder.go:355-357`: by contrast, `RunBuilder` checks
  `active.isSet()` and calls `active.signal(syscall.SIGINT)` instead of exiting.
- `specs/builder-cli-flags/assertions/exit-when-done-flag.md`, last success
  criterion: "During an active build, Ctrl+C reaches the child, and the parent
  waits for the child. ... A continuous builder loop keeps running after the
  child exits."
- `cmd/spekk/builder_loop_exit_test.go`: the only interrupted-child test
  ("once waits for interrupted child") runs `spekk builder --once`. No test sends
  Ctrl+C to `spekk loop builder` while a child is running.

## Impact

When an operator presses Ctrl+C during a `spekk loop builder` session, the
parent exits at once. The terminal also delivers SIGINT to the Claude child, but
the parent no longer waits for it, so the child can be left running with the
terminal's stdio. Its work is never committed, and the loop does not continue.
The behavior differs from `spekk builder`, which the same assertion holds to the
same contract. The commit message for #328 says Ctrl+C "still forwards to the
child and waits for it", which is true for only one of the two commands.

## Recommendation

Reopen `exit-when-done-flag` (flip it to `failed`) so the builder brings
`RunBuilderLoop` in line with `RunBuilder`. Give `launchClaude` a
`processHolder`, forward SIGINT to an active child, wait for it, and then
continue the loop, still committing the session's changes. Add a test that
interrupts `spekk loop builder` during a build. If the intended behavior is
that only `spekk builder` keeps running, edit the criterion to name that
command instead.