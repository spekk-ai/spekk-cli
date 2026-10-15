---
icon: lucide/server
---

# Sandbox Agent Architecture

How the agent client works on a sandbox, and what the control host must provide.

## Overview

The agent client is a generic Claude Code runner. It knows nothing about specs, assertions, or the spekk workflow. It accepts a prompt and an optional system prompt over a WebSocket connection, pipes them into `claude -p -`, streams the output back, and reports the result. The control host decides what to send. The agent decides only how to run it.

This split makes the agent binary useful for any task a control host wants to delegate to a remote Claude session.

## Connection model

The agent connects out to the control host. You never connect to the agent. On startup, the agent dials a WebSocket endpoint on the control host and keeps the connection open. When the connection drops, the agent reconnects with exponential backoff: 3 seconds at first, doubled on each failure, with a cap of 60 seconds. The backoff resets after a connection that lived for 10 seconds or more. A connection that ends sooner does not count, so a control host that accepts and closes at once still backs off.

A dropped connection does not end a turn. The work belongs to the agent process, not to the connection that carried its dispatch, so `claude` keeps running and the agent keeps reading its output. A frame that ends a turn waits up to 90 seconds for a live connection, and the agent sends it on whichever connection is live at that moment. This carries the report across a reconnect. A stream frame does not wait. With no connection it is dropped, because it drives a live display, and a wait on it would stall the read of the child's output.

```
┌─────────────┐         WebSocket (outbound)        ┌──────────────┐
│ Sandbox VM  │ ─────────────────────────────────▶   │ Control Host │
│ (agent)     │ ◀─────────────────────────────────   │              │
└─────────────┘         frames in both directions    └──────────────┘
```

The control host serves the WebSocket endpoint at `wss://<host>/ws/agent/`. The agent is the client. For a host that contains `localhost`, the agent uses `ws://` and `http://` instead.

## Authentication

The agent sends its token in one place: the `Authorization: Bearer <token>` header on the WebSocket upgrade request. The URL carries no token. A token in the path leaks into access logs, proxy logs, and every error string that echoes the target, so 1.18.0 removed it.

The token comes from the `SPEKK_AGENT_TOKEN` environment variable on the sandbox.

## Protocol version

The agent and the control host share one WebSocket contract: the message types, the frame fields, and the close codes. The two ship from separate repositories, so the contract carries a version number that both sides declare at connect time. This agent declares `1.1`.

1. **The client declares.** Every dial sends an `X-Spekk-Protocol: 1.1` header next to the `Authorization` header.
2. **The control host replies.** It sends a `welcome` frame that carries its own version in the `protocol` field. That version can be different from the agent's version. For example, a control host that speaks `1.0` replies to a `1.1` agent with:

```json
{
  "type":     "welcome",
  "protocol": "1.0"
}
```

3. **Each side compares the major version.** The control host enforces. The agent only informs. On the same major, the agent logs one line. On a different major, it logs a warning that names both versions and tells the operator to update the sandbox.
4. **Refusal.** When the control host refuses the agent's major version, it closes the connection with code `4004`. The agent logs one line for the operator and keeps its usual reconnect backoff. It never dials in a tight loop.

Either side can update first. An old agent sends no header, and the control host accepts it as a legacy client. A new agent against an old control host receives no `welcome` frame and continues.

**Version rules:** a change that breaks a message type, a frame field, or a close code bumps the major. An addition bumps the minor.

Version `1.1` is additive. It adds only the `terminal` field to the `result` frame (see [Turn progress](#turn-progress)). No existing frame loses a field or changes its meaning, so a `1.0` side and a `1.1` side work together. The `turn_keepalive` frame type is reserved for version `1.2`.

## Message protocol

Every frame is a JSON object with a `type` field. The `Message` struct defines the inbound shape:

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

#### Inbound (control host to agent)

| Type | Purpose | Fields |
|------|---------|--------|
| `welcome` | Declare the control host's protocol version on connect | `protocol` |
| `message` | Start or continue a Claude session | `text`, `system_prompt`, `session_id`, `agent_session_id`, `attachments` |
| `cancel` | Stop the running Claude process of a session | `agent_session_id` |
| `heartbeat_ack` | Reply to the agent's heartbeat | none |
| `error` | Report an error, for example a rejected `conversation_open` | `error`, `detail` |

#### Outbound (agent to control host)

| Type | Purpose | Fields |
|------|---------|--------|
| `stream` | One line of Claude's streaming JSON output | `data`, `agent_session_id` |
| `result` | Claude finished with exit code 0 | `session_id`, `agent_session_id`, `output`, `terminal` |
| `error` | Something went wrong | `error`, `detail`, `agent_session_id` |
| `heartbeat` | Keep-alive, every 30 seconds | none |
| `conversation_open` | Ask for a conversation with a person (from the spool) | `session_id`, `title`, `body`, `severity` |

### Routing

Every `message` frame carries an `agent_session_id`. The worker pool routes the frame to the worker for that id. Frames with the same `agent_session_id` queue on the same worker. The `session_id` field is Claude's session identifier, used for `--resume`.

## Worker pool

The agent runs a pool of five workers. Each worker handles one `agent_session_id` at a time.

- A new `agent_session_id` claims a free slot. When all five are busy, or when the session's own queue is full, the agent replies with a `capacity_exceeded` error:

```json
{
  "type":             "error",
  "error":            "capacity_exceeded",
  "detail":           "No agent worker slot is free, or this session's queue is full. Try again shortly.",
  "agent_session_id": "..."
}
```

- A message for an active `agent_session_id` joins that worker's queue, which holds 10 messages. One runner drains the queue. A follow-up starts no second runner. A full queue is refused, not waited on, because the enqueue happens under the pool lock, and a wait there would stall the dispatch of every other session.
- When a worker finishes its queue, the slot is released for a new session. The emptiness check and the release share the pool lock, so a message that arrives at the last moment is never left on a worker nobody drains.
- A `cancel` frame sends SIGTERM to the worker's running Claude process. It does not clear that session's queue, so a queued follow-up still runs. Issue [#212](https://github.com/spekk-ai/spekk-cli/issues/212) tracks this.

## Claude invocation

Each worker runs Claude as a child process:

```bash
claude -p - --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions
```

- The prompt goes to stdin. When the frame has a `system_prompt`, the agent puts it before the text, with a separator.
- When the frame has a `session_id`, the agent appends `--resume <session_id>` to continue that Claude session.
- The working directory is the configured workspace (default: `/opt/spekk/workspace`).
- Each line of stdout goes back to the control host as a `stream` frame.
- When Claude exits with code 0, the agent sends a `result` frame with the session id, the output text, and `terminal`.
- When Claude exits with an error, the agent sends an `error` frame with up to 2000 characters of stderr.

### Claude Code version

The agent runs the native Claude Code, which every provisioning path installs as the `agent` user and links to `/usr/local/bin/claude`. The service has the systemd default `PATH`, so it finds that link before an npm copy in `/usr/bin`. The minimum version is the constant `MinVersion` in `internal/claudecode`. It is the oldest version that keeps the `claude -p` process alive while a background task runs, and reports again when the task ends. An older version kills the task a few seconds after the `result` and exits, so the agent never sees how the task ended.

- At start, the agent logs the path and the version of `claude`, or a warning when the version is older than the minimum or cannot be read.
- Before each turn, the agent runs `claude --version` again. When the version is older than the minimum, or `claude` is missing, the agent does not start the turn. It sends an `error` frame whose `error` text names the version that it found, the minimum, and the remedy: run `spekk sandbox deploy`, which installs a current version.

## Turn progress

Two facts about a turn are visible only to the agent: whether the turn is still working, and whether it ended the task or only yielded. The agent reads both from structured fields of Claude's stream. It never reads the agent's words, and it never asks the agent about itself.

### `terminal` on the result frame

The `result` frame always carries `terminal`, a JSON boolean. It is `false` when a background task that the turn started was still live at the last `result` event of the Claude process, and `true` otherwise. `false` means the turn yielded: the agent gave its last word while work that it started was still live, and no later result shows that it saw how that work ended. The agent always sends the field. A control host reads an absent field as `true`, which is what an agent older than `1.1` means.

The `terminal` field needs no rollout order, because an absent field means `true`. Either side can update first.

### `turn_keepalive` (reserved for version 1.2)

The agent heartbeat is connection-scoped and says nothing about a turn, so without a turn-scoped signal a long tool call is indistinguishable from a hang. A planned `turn_keepalive` frame will fill that gap. It will carry the `agent_session_id` of the turn, and the agent will send it only while a tool call of that turn is in flight. It comes with version `1.2`.

The control host must accept and attribute `turn_keepalive` before any agent sends it. A control host that does not know a frame type logs it and replies with an `error` frame. This agent does not send `turn_keepalive`.

## Conversation spool

During a Claude session, the running process, or a tool it invokes, can ask for a conversation with a person. It writes a request file into a spool directory. This is how an agent escalates.

1. Each Claude invocation gets a private spool directory, a temporary directory.
2. The `SPEKK_CONVERSATION_SPOOL` environment variable points the process at its spool.
3. A writer creates a request file atomically: it writes `request-*.json.tmp`, then renames it to `request-*.json`. `spekk conversation open` does this.
4. The worker drains the spool after every line of Claude output, and one more time after the process exits.
5. Each valid request file becomes a `conversation_open` frame to the control host:

```json
{
  "type":       "conversation_open",
  "session_id": "claude-session-uuid",
  "title":      "Need clarification on auth flow",
  "body":       "The spec mentions OAuth but the codebase uses JWT...",
  "severity":   "info"
}
```

6. The worker removes each request file after it sends the frame. A request fires one time and is never retried.

### Request file format

```json
{
  "title":    "Short summary",
  "body":     "Detailed message",
  "severity": "info"
}
```

Valid severities: `info`, `warning`, `critical`. The default is `info`. The spool directory is removed when the Claude invocation finishes.

## Attachments

When a `message` frame has `attachments`, the agent downloads each file from the control host before it passes the prompt to Claude:

1. For each attachment, the agent sends `GET /api/agents/attachments/<id>/download/` with `Authorization: Bearer <token>`.
2. It saves the file to `<workspace>/.attachments/<timestamp>/<index>-<filename>`.
3. It appends the saved paths to the prompt text, so Claude can read the files.

## Control host requirements

A control host that works with the agent must provide:

| Requirement | Details |
|-------------|---------|
| **WebSocket endpoint** | Serve `wss://<host>/ws/agent/` and accept the upgrade with the agent's token |
| **Authentication** | Validate the `Authorization: Bearer <token>` header |
| **Protocol handshake** | Read the `X-Spekk-Protocol` header, send a `welcome` frame with its own version, and close with code `4004` on a major-version mismatch. A dial with no header is a legacy client |
| **Inbound frames** | Handle `stream`, `result`, `error`, `heartbeat`, and `conversation_open` from the agent |
| **Turn result** | Read `terminal` from the `result` frame. An absent field means `true` |
| **Outbound frames** | Send `welcome`, `message`, `cancel`, `heartbeat_ack`, and `error` to the agent |
| **Attachments** | Serve `GET /api/agents/attachments/<id>/download/` with Bearer authentication, when attachments are used |
| **Session routing** | Assign unique `agent_session_id` values and track which sessions are active |

## Registration after creation

The agent cannot connect until the control host knows its token. With `SPEKK_API_TOKEN` set, `spekk sandbox create` registers the agent through the control host API before it creates the machine, and writes the token that the control host returns to the machine. There is no manual step. `spekk sandbox destroy` revokes the agent before it deletes the machine.

Without `SPEKK_API_TOKEN`, `create` makes the token itself and prints it with a reminder to register it. Until you register it, the agent tries to connect and fails authentication, on the usual backoff. `spekk sandbox register <name>` registers it later with your API token.

`create` waits for cloud-init for `--provision-timeout` (default 30 minutes) and then stops, with the machine still running and the record at status `provisioning`. `spekk sandbox provision <name>` finishes that sandbox when `/opt/spekk/.provisioned` exists. It runs the same credential injection, git configuration, and agent deploy that `create` runs, and marks the record `active`. A registered sandbox keeps its agent token. See [`spekk sandbox`](../cli-reference.md#spekk-sandbox).

A control host that supports this registration serves `POST /api/agents/`, `POST /api/agents/<id>/revoke/`, and the schedule routes under `/api/agents/<id>/schedules/`, with `Authorization: Token <key>` authentication.

## Environment variables

The agent binary reads its configuration from `/etc/spekk/agent.env` on the sandbox:

| Variable | Required | Description |
|----------|----------|-------------|
| `SPEKK_AGENT_TOKEN` | Yes | Bearer token for the control host |
| `SPEKK_HOST` | Yes | Hostname of the control host, for example `api.example.com` |
| `WORKSPACE` | No | Working directory for Claude sessions (default: `/opt/spekk/workspace`) |

The agent sets `SPEKK_CONVERSATION_SPOOL` for each invocation. It is not in `agent.env`.

The same file carries the model credential, which the agent does not read. It passes its environment to the `claude` child process, and Claude Code reads the credential there. Which variables those are depends on the sandbox's auth mode. See [Agent runtime](../configuration.md#agent-runtime).
