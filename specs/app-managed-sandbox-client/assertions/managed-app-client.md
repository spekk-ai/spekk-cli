---
id: managed-app-client
parent: app-managed-sandbox-client
created: 2026-10-22T00:00:00Z
priority: 3
status: not_started
branch: feature/app-managed-sandbox-client
---

# One app client protects the personal key

## Success criteria

- A user sets the exact app host and stores a named personal API key from the app's Settings page in a user-only local credential file or operating-system secret store. The CLI never prints the key, writes it to a sandbox record, passes it in a process argument, or sends it to an agent. It does not copy a Claude login, cloud token, GitHub App key, shared model key, SSH key, or coordinator token. A revoke or expiry makes the next API operation fail with a clear 401 and a new-key instruction.
- The client accepts HTTPS for a remote host and plain HTTP only for loopback development. It rejects URL user info, paths, queries, fragments, and redirects before forwarding a credential. It sends `Authorization: Bearer` to that host only, bounds connect and response waits, limits error bodies, and reports method, route, HTTP status, and server reason without a credential. A redirected API or SSE request fails closed.
- The one client implements the app launch, repository search, agent, chat, and saved status routes. It does not assume that every older control-host route accepts a personal key. All operations rely on current server authentication and grants; a cached organization, repository, or agent list never authorizes a later write.
- Tests cover remote HTTP refusal, redirect refusal, key redaction, malformed host, 401, 403, 404, 409, 429, 5xx, and a dropped response. The test for a redirect proves that the target receives no Authorization header.
