---
id: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
---

# Turn Progress Reporting (Client Side)

## Problem

The control host records a lifecycle for every dispatch it sends to an agent (companion spec: the control host's `agent-session-lifecycle`). That work is done. It cannot succeed alone, because the control host sees output and not work.

Over 30 days the control host observed 35 scheduled dispatches and 6 failures. The failures are not scattered. Dispatch types that do their work synchronously never failed. The two types that start a command in the background and promise to wait failed 4 of 11 and 2 of 2. Every failure has exactly one reply and always the same shape: the agent says the command is running in the background and that it will report the summary when it finishes. The turn ends, nothing resumes it, and the summary can never arrive.

Two facts decide these cases, and only the sandbox client can see either one.

**Is this turn still working?** The control host infers progress from frames arriving. The agent-level heartbeat is no help: it is connection-scoped, fires every 30 seconds whether or not a turn is in flight, and must not advance any turn's activity. A single long tool call emits nothing at all, so genuine work and a hang look identical from outside. This was measured, not assumed: a 20.6-second `Bash` call produced **zero** stream lines between its start and its result. The control host's silence bound is 30 minutes, so any quiet stretch longer than that reads as abandoned even when the work is healthy.

**Did the turn end the task, or did the agent yield?** The control host now reads a `terminal` field from the `result` frame and treats an absent field as `true`. Today the client never sends it. Until it does, the `YIELDED` state is unreachable, and every yielding turn — the exact failure above — records as `COMPLETED` and stays invisible.

## What the client can actually observe

Assertions here are grounded in the behaviour of Claude Code 2.1.226, verified directly against the `stream-json` stream rather than inferred. Three findings shaped the design, and one of them removed the mechanism this spec was first expected to use.

**Tool calls are observable, and they bracket the silence.** A tool call starts as an `assistant` event whose `message.content[]` holds a block `{"type": "tool_use", "id": …, "name": …}`. It ends as a `user` event whose `message.content[]` holds `{"type": "tool_result", "tool_use_id": …}`. The two are paired exactly by id, so the set of started-but-unfinished ids is an exact record of what is in flight. Nothing else is emitted in between. This is the same shape as the agent heartbeat fix: absence of a signal is the signal, so a keepalive sent only during that window makes silence mean "nothing is running" rather than "nothing is printing".

**There are no top-level `tool_use` or `tool_result` events.** `internal/serve/serve.go` switches on `case "tool_use"` and `case "tool_result"` as top-level event types. A full stream capture contains neither. Those branches never execute, and the shape must not be copied into the sandbox client.

**Background work is visible as structured data, and the process tree is not.** The design intended to detect a yield by checking whether a process the agent spawned outlived the turn. That does not work, and the reason is structural: Claude Code's `Bash` tool places its shell in its own session and its own process group. Measured, claude ran at `PGID 3328047` while its own `Bash` shell ran at `PGID 3328404, SESS 3328404` as a direct child. Setting `Setpgid` on the claude process and testing the group with `kill(-pgid, 0)` therefore reports "gone" while spawned work is demonstrably still running — a background job started this way outlived claude by 35 seconds while the group read empty. Process-group liveness is not a usable signal and this spec does not use it.

What replaces it is better, because it is structured rather than inferred. Claude Code emits a live inventory of background tasks as `system` events:

| Event | Carries |
| --- | --- |
| `{"type":"system","subtype":"task_started"}` | `task_id`, `tool_use_id`, `task_type` |
| `{"type":"system","subtype":"background_tasks_changed"}` | `tasks[]` — the complete current inventory |
| `{"type":"system","subtype":"task_updated"}` | `task_id`, `patch.status` |
| `{"type":"system","subtype":"task_notification"}` | `task_id`, `status`, `output_file` |

A `tool_use` for `Bash` also carries `input.run_in_background` as a boolean, and its `tool_use_result` carries `backgroundTaskId`. None of this requires reading prose.

The failing sequence is fully legible in those events. A turn that started a 40-second background job emitted `task_started` at 4.82s, reported `result` with `subtype: success` and `terminal_reason: "completed"` at 6.03s **while the inventory still held that task**, and then Claude Code **killed** the task at 11.05s (`task_updated` with `patch.status: "killed"`, inventory back to empty). The work does not quietly continue, as the failure reports suggest — it is destroyed a few seconds after the turn claims success. The promised summary is not merely unscheduled; it is impossible.

The contrasting case resolves correctly under the same rule. A turn that started an 8-second background job emitted its first `result` with the inventory non-empty, then Claude Code waited, the task finished, and a **second** `result` event was emitted with the inventory empty. Evaluating the inventory at the **last** `result` event before the process exits gives "yielded" for the first case and "completed" for the second, which is the right answer in both.

## Decision

The client reports two things it alone can see, and nothing it has to judge.

### Progress is reported, not inferred

While at least one tool call is in flight, the client sends a turn-scoped keepalive frame. It is distinct from the agent heartbeat: it is attributed to a turn, and it is sent only when work is genuinely in flight. Silence on a turn then carries the meaning the control host's silence bound already assumes — that nothing is running.

The keepalive is a new frame type, not a synthetic `stream` frame. A `stream` frame is one line of Claude's output; manufacturing one would put text into the transcript that Claude never produced.

### Terminality is reported from the task inventory

The `result` frame carries `terminal`. It is `false` when the background-task inventory was non-empty at the last `result` event of the turn, and `true` otherwise. This is a structured observation, not an assessment, and the agent is never asked about itself.

### No self-assessment, and no string matching, anywhere

This is the constraint the whole spec exists to satisfy. The failure being replaced is a classifier over English prose. Nothing here may read the agent's words, and nothing may ask the agent whether it finished. Every input named above is a JSON field.

### The end state is that yields become impossible

Reporting a yield well is second best. The design this spec aims at is one where a turn cannot end while work it started is unfinished, so turn-end means task-end by construction and `COMPLETED` is honest without anyone trusting anything.

`terminal: false` is the transitional path, not the destination. Once it ships, a rising `YIELDED` count is itself the signal, and the count should trend to zero as the stronger behaviour lands. `turn-does-not-end-with-live-background-work` carries that goal.

One constraint bounds how it can be reached. A `claude -p` invocation is one-shot: the prompt is written to stdin, stdin is closed, and the client cannot extend a turn in place. Nor can the client stop Claude Code from killing the tasks — that happens inside claude, seconds after the result. The lever the client does have is the session resume it already uses (`--resume`), which lets it collect outstanding work before reporting a result.

### Rollout order

The control host accepts an absent `terminal` as `true`, so that half is backward compatible in the client's direction and an unchanged sandbox stays indistinguishable from one that always reports `terminal: true`.

The keepalive frame is not covered by that guarantee, and the order is not negotiable. The control host must accept and attribute a `turn_keepalive` frame **before** any client emits one. A new frame type sent to a host that does not know it is at best ignored and at worst an error, and this project has already taken one production outage — 2026-08-07, on the control host — from deploying one side of a contract change ahead of the other. The safe order is: control host accepts and attributes the frame, then the client emits it.

Both changes are additive — one new frame type, one new optional field, no change to any existing frame's meaning. The protocol major stays at `1` and the minor moves to `1.1`. No breaking change is needed and none is proposed.

## Rejected

- **Detecting a yield from the process tree.** Refuted by measurement, not by preference: Claude Code's `Bash` tool `setsid()`s its shell into its own session, so a process-group test on the claude process reports an empty group while spawned work is still running. The signal is not merely weak, it is wrong.
- **Asking the agent whether it finished.** This reintroduces self-assessment, which is the failure mode being removed. An agent that believes it will return to finish the work is exactly the agent that produced all six failures.
- **Matching the reply text.** The current control-host watchdog classifies English prose. It cannot tell "I will wait" meant as a yield from the same words quoted inside a summary, and it has no ground truth to tune against.
- **Reusing the agent heartbeat as a progress signal.** It is connection-scoped and fires whether or not a turn is running. A healthy socket is not evidence of progress; the control host's spec is explicit that a heartbeat must not advance a turn's activity.
- **Sending the keepalive on a fixed timer for the whole turn.** That makes silence meaningless again in the other direction: a hung turn would keep reporting progress forever. The keepalive must be conditional on observed in-flight work.
- **Emitting the keepalive as a synthetic `stream` frame.** It would avoid a protocol change at the cost of inserting text into the transcript that Claude never produced.
- **Making `terminal` a required field.** That strands every sandbox that has not upgraded, which is the shape of the 2026-08-07 outage.
- **A protocol major bump.** Nothing here changes an existing frame's meaning. Both changes are additive.
- **Copying the `tool_use` / `tool_result` top-level cases from `internal/serve/serve.go`.** Those event types are not emitted; the branches are dead.

## Open Questions

1. **The `system` event subtypes are not a documented API.** `task_started`, `background_tasks_changed`, `task_updated`, `task_notification`, and `tool_use_result.backgroundTaskId` were observed on Claude Code 2.1.226. They may change without notice. `claude-stream-shapes-pinned-by-fixtures` requires captured fixtures and a safe fallback so that an unrecognised stream degrades to today's behaviour rather than to a wrong verdict.
2. **Whether the control host advances turn activity on `turn_keepalive`.** The control host's spec advances `last_activity_at` on frames received for a turn, but the keepalive frame does not exist there yet. It is named as a dependency in `turn-keepalive-frame-sent-during-tool-calls`, and it is a companion change in the control host repository, not work in this one.

## Related

- The control host's `agent-session-lifecycle` spec — the server half, already implemented. It defines `DISPATCHED`, `WORKING`, `COMPLETED`, `FAILED`, `YIELDED`, and `ABANDONED`, and its formal model proves the observer's verdict is total, honest, and never rewritten.
- `specs/sandbox-protocol-version/` — the version constant and the bump rules this spec follows.
- `docs/advanced/sandbox-architecture.md` — the frame tables that record the contract.
