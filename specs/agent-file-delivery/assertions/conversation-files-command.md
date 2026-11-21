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

`spekk conversation open` can take `--attach <file>` up to three times. It sends the text and files with the session's agent token and session ID, then prints the message link and file IDs. A plain message also returns its link. The CLI refuses a file over 10 MB, files over 20 MB in total (`all files together must be at most 20 MB`), more than three files, and an absent session before sending. It accepts all file types; Slack workspace rules still apply. A partial delivery response reports that the text posted and names the cause.

The sandbox worker writes the active Claude resume ID to a private session file before the CLI sends the request. The CLI refuses a direct post when that ID is not ready, so a quick human reply uses the initiating Claude session.

The app advertises direct conversation delivery in its welcome frame. A current session with an app that does not advertise it queues a plain message through the spool. In that case, an attachment gets a clear error that names the missing app support.

**Tests:** `cmd/spekk/conversation_test.go` checks direct and spool delivery, file inputs, and partial results. `cmd/sandbox/spool_test.go` checks the app capability in the welcome frame.
