# Spekk CLI 1.12.0 — Agents Can Open Conversations

Sandbox agents are no longer mute until spoken to: an agent can now open a new conversation on the chat surface its control host bridges to, and human replies in that conversation resume the initiating agent's session.

## `spekk conversation open`

From inside a sandbox session:

```bash
spekk conversation open --title "Drift found in auth spec" \
  --body "Two done assertions contradict the shipped login flow..." \
  [--severity info|warning|critical]
```

The command writes one atomic request file to the session's private spool directory and exits. The sandbox worker drains the spool as the session streams (plus a final drain at session end), stamps the request with the authoritative session id it observed from the session's own event stream — requests can never spoof another session — and emits a `conversation_open` frame to the control host, which opens the conversation and binds it for resume.

Requests are fire-once: consumed whether the send succeeds or fails. Malformed requests, and requests arriving before any session id is known, are dropped with a log line. The writer/drainer contract (spool env var, request shape, file naming, severities) lives in one shared package — `internal/conversation` — so the two sides cannot drift.

## Worker improvements

- Inbound typed error frames (`conversation_open_invalid`, `conversation_open_no_channel`, `conversation_open_failed`) are logged legibly instead of "Unknown message type"
- The WebSocket dial now also sends the agent token as an `Authorization: Bearer` header (the path token is unchanged this release; header-only auth is a coordinated follow-up)

## Housekeeping

Neutralized infrastructure-specific wording in public files: cloud-init template comments, `sandbox create` operator guidance, and related comments now describe a generic control host.

## Upgrade

```bash
spekk update
```
