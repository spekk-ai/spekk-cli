---
id: claude-stream-shapes-pinned-by-fixtures
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
status: not_started
branch: feature/turn-progress-reporting
---

# The Claude Event Shapes This Client Depends On Are Pinned by Captured Fixtures

## Description

Everything else in this spec reads structured fields out of Claude Code's `stream-json` output. Those fields are an implementation detail of Claude Code, not a documented API, and they were observed on version 2.1.226. A silent change upstream must produce a failing test rather than a wrong verdict in production.

This assertion establishes the decoding layer and its safe fallback. Nothing here changes what the client sends.

## Success Criteria

- Captured `stream-json` fixtures live under `cmd/sandbox/testdata/`, recorded from a real `claude -p - --output-format stream-json --verbose` run, one file per scenario: a turn with a single long foreground tool call; a turn that starts a background task and ends while it is still running; a turn that starts a background task and reports a second `result` after it finishes.
- Each fixture records the Claude Code version it was captured from.
- A decoder in `cmd/sandbox` turns one raw stream line into a typed value naming only the events this client needs: a tool call starting, a tool call ending, a background-task inventory snapshot, and a turn result.
- A tool call starting is recognised as an `assistant` event carrying a `message.content[]` block of `"type": "tool_use"`, and the block's `id` is captured.
- A tool call ending is recognised as a `user` event carrying a `message.content[]` block of `"type": "tool_result"`, and the block's `tool_use_id` is captured.
- A background-task inventory snapshot is recognised as a `system` event of `subtype: "background_tasks_changed"`, and the length of its `tasks[]` array is captured.
- A turn result is recognised as a top-level `result` event.
- **Unrecognised input degrades to silence, never to a verdict.** A line that is not valid JSON, an event of an unknown `type` or `subtype`, an event missing an expected field, and a field of an unexpected JSON type each decode to "nothing observed". The decoder never returns an error that ends the turn and never panics. This is what keeps a future Claude Code release from turning a working sandbox into one that reports wrong outcomes.
- The decoder recognises **no** top-level `tool_use` or `tool_result` event type. A fixture-backed test asserts that no captured stream contains either, so the dead shape in `internal/serve/serve.go` is not reproduced here.
- Decoding is a pure function of one line plus prior state. It performs no I/O and writes no frames, so it is testable without a websocket or a claude process.
- Tests replay each fixture end to end and assert the decoded event sequence, so an upstream shape change fails the build with a message naming the event that stopped being recognised.
