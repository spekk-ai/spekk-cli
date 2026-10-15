# Claude Code stream fixtures

These files pin the Claude Code `stream-json` shapes that the sandbox client reads (`specs/turn-progress-reporting/`). `turn_test.go` replays each file. A change in a shape upstream makes a test fail and name the event that is no longer recognized. `invoke_test.go` replays the `.partial.jsonl` files through the whole client.

Each file is one complete, logged-in run of Claude Code 2.1.301, with the order of the lines unchanged:

```
claude -p - --output-format stream-json --verbose --dangerously-skip-permissions
```

The `.partial.jsonl` file of each scenario is a second run with `--include-partial-messages` added. These are the flags that the client passes to claude.

| Scenario | Prompt | What the run shows |
|------|---------|--------|
| `foreground-tool-calls` | Run `sleep 20`, `sleep 6`, and `sleep 1` as three parallel foreground `Bash` calls. | No inventory snapshot, and one `result`. |
| `background-task-stopped-at-time-limit` | Start `sleep 600` in the background, end the turn, and do not wait. | A `result` while the task is in the inventory. Claude Code keeps the process alive, stops the task at its 600-second background limit (`task_updated` with `patch.status: "killed"`, `task_notification` with `status: "stopped"`), sends an empty inventory, and starts a new turn. The agent reports the stop in a second `result`. |
| `background-task-finishes-before-second-result` | Start `sleep 15` in the background, end the turn, and reply again when it finishes. | A `result` while the task is in the inventory. The task completes, the inventory is empty, and a second `result` follows. |

## Edits

The only edits are these replacements:

- The home directory is `/home/user`, and the working directory is `/tmp/fixture-work`.
- Each `msg_` message id and `req_` request id is a fake id, such as `msg_fixture0001`. The same real id always gets the same fake id.

## A shape that these files do not show

Claude Code 2.1.226 killed a background task a few seconds after the `result` and exited with no second `result`. Claude Code 2.1.197 does the same, and it also sends no `background_tasks_changed` event. 2.1.301 does not show that shape, so no file here records it. `TestResultFrameDeclaresTerminal` tests it with a short stream that it writes itself.
