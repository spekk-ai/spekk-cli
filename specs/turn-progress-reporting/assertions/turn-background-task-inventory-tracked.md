---
id: turn-background-task-inventory-tracked
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
status: not_started
branch: feature/turn-progress-reporting
depends-on: claude-stream-shapes-pinned-by-fixtures
---

# The Client Knows How Much Background Work a Turn Has Left Running

## Description

A turn that ends while a background task is live is a yield. The client can see this without asking the agent anything and without reading a word of prose, because Claude Code publishes the inventory of running background tasks as structured `system` events.

The process tree cannot answer this question. Claude Code's `Bash` tool `setsid()`s its shell into its own session and process group, so a process-group test on the claude process reports an empty group while spawned work is still running. This assertion uses the event stream instead.

## Success Criteria

- The client maintains, per invocation, the most recent background-task inventory count, taken from the `tasks[]` array length of `background_tasks_changed` events.
- The count starts at zero for every invocation. No state from a previous turn carries over.
- The latest snapshot wins. The inventory is a full replacement, not an increment, so a missed intermediate event self-corrects on the next snapshot rather than drifting.
- The count is derived from snapshots only. It is never computed by adding starts and subtracting finishes, because a dropped event in that scheme is permanent.
- A turn that never starts a background task has a count of zero throughout.
- No inspection of prose decides the count. In particular the tool-result text that announces a background command, and any `output_file` path, are never parsed. `run_in_background` in a tool input and `backgroundTaskId` in a tool result are structured fields and may be read, but the inventory snapshot is what the count comes from.
- A fixture replay of the yielding scenario shows the count non-empty at the turn's `result` event and back to zero only after Claude Code kills the task.
- A fixture replay of the completing scenario shows the count non-empty at the first `result` event and zero at the second.
- Tests cover: no background task at all; one task started and still live at stream end; one task started and finished before stream end; a snapshot arriving with an empty `tasks[]` array after a non-empty one.
