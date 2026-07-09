---
id: idle-timeout
parent: builder-loop
created: 2026-07-09T16:00:00Z
priority: 1
status: not_started
branch: feature/advanced-loop
---

# Idle Timeout

The builder loop kills a stuck builder process after a configurable period of inactivity, preventing the loop from hanging indefinitely.

## Success Criteria

- If a builder process produces no stdout/stderr output for 120 seconds (default), the loop kills it with SIGTERM and moves to the next iteration
- The timeout duration is configurable via `--idle-timeout <seconds>` flag (default 120)
- When a timeout fires, the loop logs: `"Builder idle for 120s. Force-stopping..."` and continues to the next iteration (does not exit the loop)
- The timeout resets on any stdout or stderr activity from the builder process
- The loop must tee stdout/stderr through a pipe to detect activity while still displaying output to the user in real time
- A grace period of one poll cycle is given at startup before timeout detection begins (prevents false positives during initial prompt loading)
