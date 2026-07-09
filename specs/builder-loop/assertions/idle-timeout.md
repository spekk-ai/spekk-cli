---
id: idle-timeout
parent: builder-loop
created: 2026-07-09T16:00:00Z
priority: 1
status: in_progress
locked-by: builder-paris-macbook-pro-2.local-69311-1783632767
branch: feature/advanced-loop
---

# Idle Timeout

The builder loop kills a stuck builder process after a configurable period of inactivity, preventing the loop from hanging indefinitely. Output must stream to the user in real time via PTY allocation.

## Success Criteria

- The loop spawns Claude inside a pseudo-terminal (PTY) using `creack/pty`, so Claude behaves as if connected to a real terminal
- Claude's output streams to the user in real time (identical to running `spekk builder` directly)
- If Claude produces no output for 120 seconds (default), the loop kills it with SIGTERM and moves to the next iteration
- The timeout duration is configurable via `--idle-timeout <seconds>` flag (default 120)
- When a timeout fires, the loop logs: `"Builder idle for 120s. Force-stopping..."` and continues to the next iteration (does not exit the loop)
- The timeout resets on any stdout or stderr activity from the builder process
- A grace period of one poll cycle is given at startup before timeout detection begins (prevents false positives during initial prompt loading)
- When a builder is killed by timeout, the loop resets the assertion status from `in_progress` back to `not_started` so the next iteration can pick it up again
- Idle timeout ONLY applies to `RunBuilderLoop` — do NOT add it to `RunBuilder` (the regular `spekk builder` command)

## Constraints

**DO NOT use `cmd.StdoutPipe()` / `cmd.StderrPipe()` to monitor output.**
A previous implementation used pipes to tee stdout/stderr for activity tracking. This broke Claude Code entirely — Claude detects it is not connected to a TTY and produces no output, causing every build to false-timeout at 120s. The pipe approach was reverted in commit d87e6d7.

Use `creack/pty` to allocate a real pseudo-terminal instead. This preserves Claude's TTY detection while still allowing the loop to read output for activity tracking.
