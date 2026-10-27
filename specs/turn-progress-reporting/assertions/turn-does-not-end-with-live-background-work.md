---
id: turn-does-not-end-with-live-background-work
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 3
status: done
depends-on: result-frame-declares-terminal
---

# A Turn Never Reports Success While Work It Started Is Unfinished

## Description

Reporting a yield accurately is second best. The end state is one where a yield cannot happen: turn-end means task-end by construction, and `COMPLETED` is honest without anyone trusting anything.

**Closed (2026-10-15): Claude Code does this, and the sandbox makes sure that it runs a version that does.** See `specs/sandbox-claude-code-version/`. From 2.1.292, a `claude -p` process does not exit while a background task runs. It waits until the task ends or reaches its 600-second limit, starts a new turn, and the agent sends a later `result` with an empty inventory. 2.1.291 and older versions kill the task seconds after the `result` and exit. The measurements are in the parent spec of `sandbox-claude-code-version`, and the 2.1.301 runs are recorded in `cmd/sandbox/testdata/` (`background-task-finishes-before-second-result`, `background-task-stopped-at-time-limit`).

The client mechanism that this assertion first described (resume the session until the inventory is empty, with a bound) is not built, and it is not needed. Its two design gaps (what prompt the resumed turn gets, and a killed command with side effects that runs two times) do not occur, because Claude Code continues the turn itself. The first version of this assertion is in the git history before this change.

## Success Criteria

- The sandbox client starts no turn on a Claude Code version older than `claudecode.MinVersion` (`client-refuses-turn-below-minimum-claude-code`), and that minimum is a version that keeps `claude -p` alive while a background task runs.
- The client adds no mechanism of its own: it does not resume a session to wait for a task, and it adds no text to the prompt.
- `terminal: false` remains implemented and remains correct (`result-frame-declares-terminal`). On a version at or above the minimum, the last `result` has an empty inventory in every recorded run, so `terminal` is `true`.
