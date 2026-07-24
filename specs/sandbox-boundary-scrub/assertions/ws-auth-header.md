---
id: ws-auth-header
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 1
status: not_started
---

# Agent Auth Token Travels in an Authorization Header

`cmd/sandbox/client.go`'s `wsURL()` embeds the agent auth token in the
WebSocket URL path (`wss://<host>/ws/agent/<token>/`). Tokens in URLs leak into
logs, proxies, and error messages. The token should be presented in an
`Authorization` header on the dial instead.

## Success Criteria

- The dial in `connect()` sets an `Authorization` header carrying the token
  (e.g. `Authorization: Bearer <token>`) on the `websocket.DialOptions`
  (`HTTPHeader`), so the token is sent as a header rather than only in the URL.
- The token value is no longer required to be in the URL path for
  authentication: the header is the authoritative carrier.
- **Coordination note (cross-repo, must be honored):** the control host today
  authenticates the agent from the *path* token in its `/ws/agent/<token>/`
  route. Until the control host is updated to read the `Authorization` header,
  removing the path token outright would break auth. So this change is
  **additive**: send the header now, and keep the path form working as a
  deprecated fallback. Once the control host reads the header, a follow-up
  removes the token from the path so it stops appearing in URLs. Whether the
  control host already reads the header is the one thing to confirm before
  deciding to drop the path token in this same change.
- If, at build time, the control host is confirmed to read the header, the path
  token may be dropped in this change and `wsURL()` reduced to
  `wss://<host>/ws/agent/`; otherwise the path token stays as the deprecated
  fallback and this is noted in a comment at `wsURL()`.
- The `ws` vs `wss` scheme selection (localhost → `ws`) is preserved.
- A test asserts the dial is invoked with an `Authorization` header whose value
  contains the token.
