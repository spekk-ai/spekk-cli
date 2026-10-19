---
id: chatgpt-cache-over-stdin
parent: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
status: done
branch: sc1-codex-sandboxes
---

# Optional ChatGPT cache installs as a private file

## Success Criteria

- If set, create and provision check that the cache is a valid private file before touching a machine. Subscription creation can proceed without this file.
- The cache reaches the sandbox only through SSH stdin. The install refuses unsafe existing paths and atomically replaces the cache with an agent-owned 0600 regular file.
- Destroying an unmanaged machine removes that cache.
- No token appears in arguments, output, or logs.

## Verification

- The invalid-cache and unsafe-source tests check preflight. Existing credential boundary and teardown tests check transport and removal. A real sandbox install remains unverified.
