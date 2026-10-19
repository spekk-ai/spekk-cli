---
id: chatgpt-cache-over-stdin
parent: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
status: done
branch: sc1-codex-sandboxes
---

# Subscription creation installs a private ChatGPT login cache

## Success Criteria

- Create and provision check for a valid file-backed ChatGPT login cache before touching a machine.
- The cache reaches the sandbox only through SSH stdin and lands at `/home/agent/.codex/auth.json` with agent ownership and mode 0600.
- Destroying an unmanaged machine removes that cache.
- No token appears in arguments, output, or logs.

## Verification

- A fake SSH test checks the stdin boundary and private install script. Existing teardown inventory test checks removal.
