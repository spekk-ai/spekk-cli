---
id: websocket-origin-validation
parent: golang-security-hardening
created: 2026-04-24T12:00:00Z
priority: 1
status: done
branch: feature/golang-migration
---

# WebSocket server restricts connection origins to localhost

The `spekk serve` WebSocket upgrader validates the `Origin` header and only accepts connections originating from localhost (`http://localhost:*`, `http://127.0.0.1:*`, `http://[::1]:*`, and Chrome extension origins). All other origins are rejected with HTTP 403.

## Success Criteria

- `upgrader.CheckOrigin` validates the request Origin header against a localhost allowlist
- Chrome extension origins (`chrome-extension://`) are also accepted
- Connections with no Origin header are accepted (non-browser clients like CLI tools)
- Connections from any other origin (e.g. `http://evil.com`) are rejected
- Test covers accepted and rejected origins
