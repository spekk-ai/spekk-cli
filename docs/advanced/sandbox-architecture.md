---
icon: lucide/server
---

# Sandbox Agent Architecture

How the sandbox agent binary works once deployed, and what the control host must provide.

---

## Overview

The sandbox agent is a **generic Claude Code runner**. It is not spec-aware — it knows nothing about specs, assertions, or the spekk workflow. It accepts a text prompt and an optional system prompt over a WebSocket connection, pipes them into `claude -p -`, streams the output back, and reports the final result. The control host decides *what* to send; the agent only decides *how* to run it.

This separation means the agent binary is reusable for any task a control host wants to delegate to a remote Claude session.

---

## Connection model

The agent connects **out** to the control host — you never connect to the agent directly. On startup, the agent dials a WebSocket endpoint on the control host and holds the connection open indefinitely. If the connection drops, the agent reconnects with exponential backoff (3 s base, 60 s cap). The backoff resets after a connection that **lived** — one that ends within 10 seconds does not count, so a control host that accepts and closes at once still backs off.

**A dropped connection does not end a turn.** The work belongs to the agent process, not to the connection that carried its dispatch, so `claude` keeps running and its output keeps being read. A frame that ends a turn waits up to 90 seconds for a live connection and is delivered on whichever one is live when it is sent, which carries the report across a reconnect. Stream frames do not wait: with no connection they are dropped, because they drive a live display and blocking on them would stall the read of the child's output.

```
┌─────────────┐         WebSocket (outbound)        ┌──────────────┐
│ Sandbox VM  │ ─────────────────────────────────▶   │ Control Host │
│ (agent)     │ ◀─────────────────────────────────   │              │
└─────────────┘         frames in both directions    └──────────────┘
```

The control host serves the WebSocket endpoint. The agent is the client.

---

## Authentication

The agent authenticates with its token in two ways (both sent on every connection):

1. **Authorization header** — `Authorization: Bearer <token>` on the WebSocket upgrade request.
2. **Path token** (legacy) — the token is embedded in the WebSocket URL path: `wss://<host>/ws/agent/<token>/`. This path-based auth is being phased out in favor of the header; both are sent during the transition.

The token is read from the `SPEKK_AGENT_TOKEN` environment variable on the sandbox VM.

---

## Protocol version

The agent and the control host share one WebSocket contract — message types, frame fields, and close codes. The two ship from separate repositories, so the contract carries a version number that both sides declare at connect time. The version starts at `1.0`.

1. **Client declares** — every dial sends an `X-Spekk-Protocol: 1.0` header next to the `Authorization` header.
2. **Control host replies** — the control host sends a `welcome` frame that carries its own version in the `protocol` field:

```json
{
  "type":     "welcome",
  "protocol": "1.0"
}
```

3. **Each side compares the major version.** The control host enforces; the agent only informs. On the same major, the agent logs one line. On a different major, it logs a warning that names both versions and tells the operator to update the sandbox.
4. **Refusal** — if the control host refuses the agent's major version, it closes the connection with code `4004`. The agent logs one operator-facing line and keeps its normal reconnect backoff. It never hot-loops.

Either side can update first. An old agent sends no header, and the control host accepts it as a legacy client. A new agent against an old control host receives no `welcome` frame and continues.

**Bump rules:** a breaking change to message types, frame fields, or close codes bumps the major. An additive change bumps the minor.

---

## Message protocol

All frames are JSON objects with a `type` field. The `Message` struct defines the inbound shape:

```json
{
  "type":             "message",
  "text":             "the prompt text",
  "system_prompt":    "optional system prompt",
  "session_id":       "resume-session-uuid",
  "agent_session_id": "routing-key",
  "attachments":      [{"id": "uuid", "filename": "spec.md", "mimetype": "text/markdown"}],
  "error":            "",
  "detail":           "",
  "protocol":         ""
}
```

### Frame types

#### Inbound (control host → agent)

| Type | Purpose | Key fields |
|------|---------|------------|
| `welcome` | Announce the control host's protocol version on connect | `protocol` |
| `message` | Start or continue a Claude session | `text`, `system_prompt`, `session_id`, `agent_session_id`, `attachments` |
| `cancel` | Terminate the running Claude process for a session | `agent_session_id` |
| `heartbeat_ack` | Response to the agent's heartbeat | *(none)* |
| `error` | Report an error (e.g. rejected `conversation_open`) | `error`, `detail` |

#### Outbound (agent → control host)

| Type | Purpose | Key fields |
|------|---------|------------|
| `stream` | One line of Claude's streaming JSON output | `data` |
| `result` | Claude finished successfully | `session_id`, `agent_session_id`, `output` |
| `error` | Something went wrong | `error`, `detail`, `agent_session_id` |
| `heartbeat` | Keep-alive sent every 30 seconds | *(none)* |
| `conversation_open` | Request a human conversation (from the spool) | `session_id`, `title`, `body`, `severity` |

### Routing

Every `message` frame carries an `agent_session_id` that the worker pool uses to route messages to the correct worker. Multiple messages with the same `agent_session_id` are queued on the same worker. The `session_id` field is Claude's session identifier, used for `--resume`.

---

## Worker pool

The agent runs a pool of **5 concurrent workers**. Each worker handles one `agent_session_id` at a time.

- A new `agent_session_id` claims an available slot. If all 5 are busy, or if the session's own queue is full, the agent replies with a `capacity_exceeded` error:

```json
{
  "type":             "error",
  "error":            "capacity_exceeded",
  "detail":           "No agent worker slot is free, or this session's queue is full. Try again shortly.",
  "agent_session_id": "..."
}
```

- Messages with an already-active `agent_session_id` are enqueued on that worker (buffer of 10). One runner drains that queue; a follow-up starts no second runner. A full queue is refused rather than waited on, because the enqueue happens under the pool lock and blocking there would stall every other session's dispatch.
- When a worker finishes its queue, the slot is released for new sessions. The emptiness check and the release share the pool lock, so a message that arrives at the last moment is never left on a worker nobody is draining.
- A `cancel` frame sends SIGTERM to the worker's running Claude process. It does not clear that session's queue, so a queued follow-up still runs (see issue #212).

---

## Claude invocation

Each worker invokes Claude as a subprocess:

```bash
claude -p - --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions
```

- The prompt is written to stdin. If a `system_prompt` is provided, it is prepended to the text with a separator.
- If `session_id` is provided, `--resume <session_id>` is appended to resume an existing Claude session.
- The working directory is the configured workspace (default: `/opt/spekk/workspace`).
- stdout is streamed line-by-line back to the control host as `stream` frames.
- When Claude exits cleanly, a `result` frame is sent with the final session ID and output text.
- When Claude exits with an error, an `error` frame is sent with up to 2000 characters of stderr.

---

## Conversation spool

During a Claude session, the running process (or tools it invokes) can request a human conversation by writing a request file to a **spool directory**. This is how agents escalate to humans.

### How it works

1. Each Claude invocation gets a private spool directory (a temp dir).
2. The `SPEKK_CONVERSATION_SPOOL` environment variable points the process at its spool.
3. A writer creates a request file atomically: write to `request-*.json.tmp`, then rename to `request-*.json`.
4. The worker drains the spool after every line of Claude output and once more after the process exits.
5. Each valid request file becomes a `conversation_open` frame sent to the control host:

```json
{
  "type":       "conversation_open",
  "session_id": "claude-session-uuid",
  "title":      "Need clarification on auth flow",
  "body":       "The spec mentions OAuth but the codebase uses JWT...",
  "severity":   "info"
}
```

6. Request files are removed after processing — **fire-once** semantics. A request is never retried.

### Request file format

```json
{
  "title":    "Short summary",
  "body":     "Detailed message",
  "severity": "info"
}
```

Valid severities: `info`, `warning`, `critical`. Defaults to `info` if omitted. The spool directory is cleaned up when the Claude invocation finishes.

---

## Attachments

When a `message` frame includes `attachments`, the agent downloads each file from the control host before passing the prompt to Claude:

1. For each attachment, the agent makes an authenticated GET request:
   `GET /api/agents/attachments/<id>/download/` with `Authorization: Bearer <token>`.
2. Files are saved to `<workspace>/.attachments/<timestamp>/<index>-<filename>`.
3. The prompt text is appended with the saved file paths so Claude can read them.

---

## Control host requirements

To work with the sandbox agent, a control host must implement:

| Requirement | Details |
|-------------|---------|
| **WebSocket endpoint** | Serve `wss://<host>/ws/agent/<token>/` accepting the upgrade with the agent's token |
| **Auth validation** | Validate the `Authorization: Bearer <token>` header (and/or the path token during transition) |
| **Protocol handshake** | Read the `X-Spekk-Protocol` header, send a `welcome` frame with its own version, and close with code `4004` on a major-version mismatch. A dial with no header is a legacy client. |
| **Inbound frame handling** | Process `stream`, `result`, `error`, `heartbeat`, and `conversation_open` frames from the agent |
| **Outbound frame sending** | Send `welcome`, `message`, `cancel`, `heartbeat_ack`, and `error` frames to the agent |
| **Attachment serving** | Serve `GET /api/agents/attachments/<id>/download/` with Bearer auth (only if attachments are used) |
| **Agent session routing** | Assign unique `agent_session_id` values and track which sessions are active |

---

## Post-creation registration

After `spekk sandbox create` provisions the VM and deploys the agent binary, one manual step remains: **register the agent's token in the control host**. The agent cannot connect until the control host recognizes its token.

The `create` command prints the token and a reminder to register it. Until registration, the agent will repeatedly attempt to connect and fail authentication.

---

## Environment variables

The agent binary reads its configuration from environment variables, typically stored in `/etc/spekk/agent.env` on the sandbox VM:

| Variable | Required | Description |
|----------|----------|-------------|
| `SPEKK_AGENT_TOKEN` | Yes | Bearer token for authenticating with the control host |
| `SPEKK_HOST` | Yes | Hostname of the control host (e.g. `api.example.com`) |
| `WORKSPACE` | No | Working directory for Claude sessions (default: `/opt/spekk/workspace`) |

The `SPEKK_CONVERSATION_SPOOL` variable is set per-invocation by the agent itself — it is not configured in `agent.env`.

The same file also carries the model credential, which the agent does not read itself: it passes its environment to the `claude` child process, and Claude Code reads it there. Which variables those are depends on the sandbox's auth mode — see [Configuration](../configuration.md#agent-runtime).
