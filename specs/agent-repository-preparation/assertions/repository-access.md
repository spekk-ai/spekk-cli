---
id: agent-chat-repository-access
parent: agent-repository-preparation
created: 2026-10-19T20:40:00Z
priority: 1
status: done
branch: feature/repository-chat-preparation
---

# Repository access belongs to the dispatched chat turn

**Tests:** cmd/sandbox/repository_test.go

## Success Criteria

- An optional `repository` object in a `message` frame names a repository ID and selected ref. A frame without it follows the old chat path.
- Before a repository turn starts Claude, the client asks the control host for the canonical repository name, selected ref, and a short lived GitHub token. The request uses the agent token in an Authorization header and identifies the dispatched agent session. The client requires an approved answer for the selected ref on every turn.
- A refused or malformed access answer ends the turn with a clear error. No repository command runs after refusal.
- Git receives a credential through its helper. A credential is absent from a clone URL, command argument, log, and error frame. The helper gives a credential only for the approved GitHub repository path.

**Companion server hook:** `POST /api/agents/{agent_id}/repository-access/` accepts `agent_session_id`, `repository_id`, and `ref`; its answer has `repository`, `ref`, and `token`. The app must authorize the linked chat message, the person, the repository, and the selected ref before it answers. The app sends the optional frame object only for a chat message that requests a repository. This app work remains open in PR #448.
