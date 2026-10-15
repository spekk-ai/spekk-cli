# Claude Code stream fixtures

These files pin the Claude Code `stream-json` shapes that the sandbox client reads (`specs/turn-progress-reporting/`). `turn_test.go` replays each file. A change in a shape upstream makes a test fail and name the event that is no longer recognized.

Every line is a real line from Claude Code 2.1.301. The first line of each file is the `init` event, and its `claude_code_version` field records the version. A new capture must keep this rule.

| File | Scenario |
|------|----------|
| `foreground-tool-calls.jsonl` | A 20-second foreground `Bash` call, with two shorter calls that overlap it and end in reverse order. Nothing comes between the start and the end of the long call, other than its own `task_started` and `task_notification`. |
| `background-task-outlives-turn.jsonl` | A background task that is still in the inventory at the `result` event. The task is killed after the result. |
| `background-task-finishes-before-second-result.jsonl` | A background task that finishes after the first `result` event, and a second `result` event with an empty inventory. |

## How the files were made

The build machine had no Claude login, so it could not run a complete `claude -p` turn. The lines come from two sources on that machine:

- The tool call, tool result, and `system` task lines come in their original order from a logged-in Claude Code 2.1.301 session that ran with `--output-format stream-json --verbose`. The commands in them are neutral `sleep` commands.
- The `init` and `result` lines come from `claude -p - --output-format stream-json --verbose` runs of the same binary without a login. Their `result` text is the login error. The client reads only the `type` of a `result` event, so this does not change what the tests prove. The two `result` lines in the last file come from two different runs, so their `session_id` values are different.

The only edit is the home directory: `/home/<user>` is replaced with `/home/user`.

These captures do not include `--include-partial-messages`, which the client passes to claude. That flag adds `stream_event` lines. The decoder does not read them, and a test asserts that such a line observes nothing.
