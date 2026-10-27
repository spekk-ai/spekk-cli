---
id: app-managed-sandbox-client
created: 2026-10-22T00:00:00Z
priority: 3
---

# A local agent uses a managed sandbox through the app

A developer can call a local CLI from Claude Code or another agent to launch a managed sandbox, send a text task, and collect its result. The CLI calls the app's launch and chat APIs. It does not run direct cloud provisioning. The present `spekk sandbox` commands still manage machines and require local provider or machine credentials; `spekk conversation open` is for a process inside a sandbox. The new command family must identify app-managed resources in its name and output.

The app must first provide stable request IDs for launch and chat, as specified by its local-agent-managed-sandboxes contract. The repository-free path must wait for the app launch-page worker gate. Keep this queue below the active launch-page and attachment work. The independent Codex login prototype is outside this contract.

## Assertions

- [managed-app-client](assertions/managed-app-client.md) provides one private and safe authenticated app client.
- [managed-sandbox-commands](assertions/managed-sandbox-commands.md) gives a recoverable launch and task workflow.
- [managed-client-guide](assertions/managed-client-guide.md) documents a cold-user setup and acceptance run.
- [managed-commands-acceptance-run](assertions/managed-commands-acceptance-run.md) proves the workflow in a cold run against a local app with a fake cloud.
