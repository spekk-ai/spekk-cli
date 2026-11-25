---
id: agent-current-thread-reply
parent: agent-file-delivery
created: 2026-11-19T23:32:00Z
priority: 1
status: done
branch: feature/agent-thread-reply
---

# Agent current thread reply

The CLI can send text and zero to three files for the current Slack thread. Before it sends, it refuses a file over 10 MB and files over 20 MB in total (`all files together must be at most 20 MB`). Its request sends the current session ID, turn ID, and Claude resume ID, without a channel or thread timestamp. If the welcome frame lacks the current thread capability, the CLI returns a clear error and sends no request. If the current turn has no turn ID (a turn that did not come from Slack), the CLI sends no request and fails with an error that says the turn has no current Slack thread and names `spekk conversation open` instead; the error does not ask the agent to try again, because the turn ID cannot appear later in the same turn. A partial result names the posted message. A second `reply` call after that result posts the text again.
