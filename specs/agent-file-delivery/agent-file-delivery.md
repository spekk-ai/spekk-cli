---
id: agent-file-delivery
created: 2026-11-19T00:00:00Z
priority: 1
---

# Agent file delivery

The installed CLI lets a sandbox agent publish an artifact or open a Slack conversation with files. It uses the agent token of its session and reports the result to the agent.

## Text artifact extension (2026-11-20)

The upload command can infer `text` for safe text and code files and accepts `--kind text`. A server without the text kind gives a clear compatibility error.
