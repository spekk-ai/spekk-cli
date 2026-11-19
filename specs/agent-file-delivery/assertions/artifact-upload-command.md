---
id: artifact-upload-command
parent: agent-file-delivery
created: 2026-11-19T00:00:00Z
priority: 1
status: not_started
branch: feature/agent-file-delivery
---

# Artifact upload command

`spekk artifact upload <file> [--title <text>] [--kind <kind>]` uses `SPEKK_HOST` and `SPEKK_AGENT_TOKEN` to upload to the existing agent artifact endpoint. It prints the app link. A 404 gives a clear message that artifacts are off. Kind defaults from a known file extension. The command sends no access field, so the server keeps organization access.

**Tests:** A local HTTP server checks a successful multipart upload and the off response. A CLI check confirms the result and no token in output.
