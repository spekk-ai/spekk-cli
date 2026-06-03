---
id: websocket-requires-session-nonce
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 2
status: done
depends-on: sandbox-name-validated
branch: feature/spekk-sandbox-vulnrabilities
---

# WebSocket connection requires a session nonce

The `spekk serve` WebSocket endpoint requires a per-session nonce token to authenticate connections. The nonce is generated when the server starts and must be provided as a query parameter when connecting. This prevents any local process from connecting to the WebSocket without knowing the nonce.

## Success Criteria

- Server generates a cryptographically random nonce on startup (e.g., 32-byte hex string)
- The nonce is printed to stdout or passed to the launched browser as a URL parameter
- WebSocket upgrade requests without the correct nonce in the query string are rejected with HTTP 403
- The HTML page served by `spekk show` includes the nonce in its WebSocket connection URL
- Empty Origin requests without a valid nonce are rejected (closing the bypass)
- Tests cover connections with valid nonce, missing nonce, and wrong nonce
