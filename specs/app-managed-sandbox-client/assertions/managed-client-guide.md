---
id: managed-client-guide
parent: app-managed-sandbox-client
created: 2026-10-22T00:00:00Z
priority: 3
status: not_started
branch: feature/app-managed-sandbox-client
depends-on: managed-sandbox-commands
---

# A new developer can use the workflow from the guide

## Success criteria

- The guide starts with app sign-in, Settings > API keys > Create key, a name and expiry, one-time copy, private local storage, correct app host, revoke, and replacement after expiry. It then shows the command flow from permitted options through explicit cleanup and a fresh HOME terminal transcript. It distinguishes app-managed commands from direct `spekk sandbox` cloud commands and from `spekk conversation open` inside a sandbox.
- The guide says that a team sandbox needs an organization model key and an admin launcher, while a personal sandbox needs its owner's remote Claude Code sign-in. It explains organization membership, agent grants, GitHub push, cloud funding, quotas, expiry, launch flag, and repository-free flag as server decisions. It never presents app admin as GitHub push access.
- The guide gives recovery steps by launch request ID and task request or message ID for a client restart, HTTP timeout, dropped stream, and failed launch. It states that stopping a local command does not stop a remote task or billable sandbox. It names the owner of cleanup after each failure and includes an explicit cancel or destroy step. It shows that text tasks work without attachments or artifacts, and lists those as conditional additions when their separate contracts and deployment are ready.
- An MCP evaluation may add a thin adapter over the same app client, with tools for options, repository search, launch and status, send and saved result, and cleanup. Each long task returns stable IDs promptly. The adapter adds no provider, policy layer, or indefinite MCP wait. REST and existing SSE are the first transport; WebSocket is required only after a documented behavior cannot use them. No MCP server is required for the first CLI release.
