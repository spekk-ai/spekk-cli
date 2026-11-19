---
id: agent-current-thread-reply
parent: agent-file-delivery
created: 2026-11-19T23:32:00Z
priority: 1
status: not_started
branch: feature/agent-thread-reply
---

# Agent current thread reply

The CLI can send text and zero to three files for the current Slack thread. Its request sends the current session ID and Claude resume ID, without a channel or thread timestamp. If the welcome frame lacks the current thread capability, the CLI returns a clear error and sends no request. A partial result names the posted message so the agent does not send the text again.
