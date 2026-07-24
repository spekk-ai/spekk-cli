---
id: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
---

# Sandbox Conversation Open

## Overview

A Claude session running inside a sandbox can already stream results back to
the orchestration server (the "control host") over the agent WebSocket
connection. It has no way to ask the control host to **open a new
conversation** — for example to escalate for human input, flag a finding, or
hand a decision back to a person on the control host's chat surface.

The control host already supports this on its side. On receiving a
`conversation_open` frame from the worker it opens a new conversation on its
chat surface, binds it to the originating Claude session, and routes any human
replies back into that session as a resume. This spec covers the **worker/
sandbox half**: emitting well-formed `conversation_open` frames, giving the
agent inside the session a way to trigger one, and logging the control host's
typed rejections legibly.

## The frame contract (control host, already built)

Worker → control host, over the existing agent WebSocket:

```json
{
  "type": "conversation_open",
  "session_id": "<the initiating Claude session id — required>",
  "title": "<short headline>",
  "body": "<longer text>",
  "severity": "info | warning | critical",
  "metadata": { "...optional...": "..." }
}
```

Control host → worker on rejection (a normal `type: "error"` frame):

```json
{ "type": "error",
  "error": "conversation_open_invalid | conversation_open_no_channel | conversation_open_failed",
  "detail": "..." }
```

## The trigger mechanism (chosen design)

The agent inside the session is the `claude` process the worker spawns in
`cmd/sandbox/invoke.go`. The only channel out of that process today is its
stdout stream-json, which the worker parses line by line — but the agent
cannot inject arbitrary top-level frames into that stream, so parsing it for a
trigger marker would be fragile.

Instead, the trigger is a small **CLI subcommand the agent runs as a tool**,
backed by a **per-session spool directory the worker drains**:

- The worker creates a private spool directory for each session and points the
  spawned `claude` process at it via an environment variable set on that one
  command (`cmd.Env = append(os.Environ(), …)` in `invoke.go`), so the variable
  is scoped to that process — not the whole worker — and the agent's shell tools
  inherit it alongside the rest of the worker's environment.
- `spekk conversation open --title … --body … [--severity …]` writes one
  request file into that spool directory (atomically) and exits. It does **not**
  supply a session id.
- The worker drains the spool during the session and emits one
  `conversation_open` frame per request on the WebSocket connection, **stamping
  `session_id` itself** from the session's own event stream. The agent cannot
  set or spoof the session id — the worker is the authority, exactly as it
  already is for the `result` frame's `session_id`.

This was chosen over the alternatives because it is the least fragile fit for
the existing plumbing: no new long-lived socket or listener, no dependency on
parsing nested stream-json content, no MCP server to configure, and a
discoverable, validated agent-facing surface (`spekk` is already the CLI
present in the sandbox). A raw sentinel file written by the agent by hand was
rejected because it gives the agent no validation and an undocumented contract;
the thin CLI wrapper is the boring, self-documenting interface.

## The request-file contract (shared source of truth)

Both the CLI (writer) and the worker (drainer) live in separate `main` packages
(`cmd/spekk` and `cmd/sandbox`), so the contract they share — the env-var name,
the request-file JSON shape, and the severity set — lives in a single shared
internal package (`internal/conversation`) that both import, rather than being
re-declared on each side where the two copies could drift. They must agree on:

- Location: the directory named by the environment variable the worker sets on
  the spawned process. Its name is the shared package's env-var constant
  (`SPEKK_CONVERSATION_SPOOL`), declared once and imported everywhere.
- One request per file. The file is a JSON object carrying `title`, `body`, and
  `severity` only — **never** `session_id` (the worker stamps that).
- Files are written atomically (write-then-rename) so the worker never reads a
  partial file.
- The worker deletes each file after emitting its frame.

## Scope discipline

This is a single message type in a single direction. Explicitly **out of
scope**: any general agent↔worker RPC framework; retry/queue/offline buffering
(emit once, log failure); multi-surface abstractions in the worker (the frame
is already the neutral abstraction); configurable severity levels beyond the
three; and any change to agent prompts (an agent composing this into its own
behavior is a later cycle).

## Assertions

1. `conversation-open-contract` — shared `internal/conversation` package: the
   env-var name, request-file struct, and severity set both binaries import
2. `conversation-open-frame` — the frame shape and its encoding constraints
3. `conversation-open-cli` — the `spekk conversation open` agent trigger
4. `worker-emits-conversation-open` — the worker drains the spool and emits
5. `conversation-open-error-frames` — inbound typed rejections are logged
