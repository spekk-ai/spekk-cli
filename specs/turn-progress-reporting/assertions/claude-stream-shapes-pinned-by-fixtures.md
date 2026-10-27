---
id: claude-stream-shapes-pinned-by-fixtures
parent: turn-progress-reporting
created: 2026-08-07T23:30:00Z
priority: 1
status: done
branch: feat/turn-progress-client
---

# The Claude Event Shapes This Client Depends On Are Pinned by Captured Fixtures

## Description

Everything else in this spec reads structured fields out of Claude Code's `stream-json` output. Those fields are an implementation detail of Claude Code, not a documented API, and they were observed on version 2.1.226. A silent change upstream must produce a failing test rather than a wrong verdict in production.

This assertion establishes the decoding layer and its safe fallback. Nothing here changes what the client sends.

## Success Criteria

- Captured `stream-json` fixtures live under `cmd/sandbox/testdata/`. Each file is one complete, logged-in `claude -p --output-format stream-json --verbose` run, with its lines in their original order, one file per scenario: a turn with a long foreground tool call; a turn that starts a background task that outlives the first `result`; a turn that starts a background task and reports a second `result` after it finishes. Each scenario is recorded a second time with `--include-partial-messages`, because the client passes that flag. `cmd/sandbox/testdata/README.md` records the command, the prompt, and the edits.
- The only edits to a fixture are fake values for home paths, the working directory, `request_id`, and message ids.
- Each fixture records the Claude Code version it was captured from, in its `init` event.
- A decoder in `cmd/sandbox` turns one decoded stream line into a typed value naming only the events this client needs: a background-task inventory snapshot and a turn result. Each stream line is decoded as JSON once, in `invoke`, and the decoder reads that value.
- A background-task inventory snapshot is recognised as a `system` event of `subtype: "background_tasks_changed"`, and the length of its `tasks[]` array is captured.
- A turn result is recognised as a top-level `result` event.
- **Unrecognised input degrades to silence, never to a verdict.** A line that is not valid JSON, an event of an unknown `type` or `subtype`, an event missing an expected field, and a field of an unexpected JSON type each decode to "nothing observed". The decoder never returns an error that ends the turn and never panics. This is what keeps a future Claude Code release from turning a working sandbox into one that reports wrong outcomes.
- The decoder recognises **no** top-level `tool_use` or `tool_result` event type, so the dead shape in `internal/serve/serve.go` is not reproduced here.
- Decoding is a pure function of one decoded line. It performs no I/O and writes no frames, so it is testable without a websocket or a claude process.
- Tests replay each fixture end to end and assert the decoded event sequence, so an upstream shape change fails the build with a message naming the event that stopped being recognised.

**Tests:** `cmd/sandbox/turn_test.go`

**Note (2026-10-15).** The decoding of tool-call starts and ends moved to `turn-inflight-tool-calls-tracked`, because only the keepalive reads it. The first fixtures were assembled from lines of several runs. They were replaced with complete runs.
