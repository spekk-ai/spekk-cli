---
id: agent-file-delivery
created: 2026-11-19T00:00:00Z
priority: 1
---

# Agent file delivery

The installed CLI lets a sandbox agent publish an artifact or open a Slack conversation with files. It uses the agent token of its session and reports the result to the agent.

## Text artifact extension (2026-11-20)

The upload command can infer `text` for safe text and code files and accepts `--kind text`. A server without the text kind gives a clear compatibility error.

## Current thread reply and the worker handoff (2026-11-27)

The CLI can also reply in the Slack thread of the current sandbox turn (`spekk conversation reply`).

The CLI cannot read the protocol frames, so the sandbox worker passes what it knows to the Claude child as environment variables. It sets them once, when the turn starts:

- `SPEKK_AGENT_SESSION_ID`: the `agent_session_id` of the `message` frame.
- `SPEKK_AGENT_TURN_ID`: the `agent_turn_id` of the `message` frame. It is empty for a turn that did not come from Slack.
- `SPEKK_CONVERSATION_DELIVERY`: `1` when the welcome frame advertised `agent_conversation_delivery`, otherwise `0`.
- `SPEKK_CURRENT_THREAD_DELIVERY`: `1` when the welcome frame advertised `agent_current_thread_delivery`, otherwise `0`.

The CLI makes each delivery decision from these variables. When an assertion says "the welcome frame lacks a capability" or "the turn has no turn ID", the CLI reads that from these variables.
