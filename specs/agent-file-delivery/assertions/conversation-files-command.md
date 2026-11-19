---
id: conversation-files-command
parent: agent-file-delivery
created: 2026-11-19T00:00:00Z
priority: 2
status: done
branch: feature/agent-file-delivery
depends-on: artifact-upload-command
---

# Conversation files command

`spekk conversation open` can take `--attach <file>` up to three times. It sends the text and files with the session's agent token and session ID, then prints the message link and file IDs. A plain message also returns its link. The CLI refuses a file over 10 MB, more than three files, and an absent session before sending. It accepts all file types; Slack workspace rules still apply. A partial delivery response reports that the text posted and names the cause.

The sandbox worker writes the active Claude resume ID to a private session file before the CLI sends the request. The CLI refuses a direct post when that ID is not ready, so a quick human reply uses the initiating Claude session.

**Tests:** A local HTTP server checks a plain message, multiple files, and a partial delivery response. Invalid file inputs make no request.
