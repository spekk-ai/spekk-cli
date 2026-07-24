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
  **Note:** sending this header is safe even before the control host reads it —
  a WebSocket server does not reject a handshake for carrying an unrecognized
  header (it validates `Sec-WebSocket-*` and its own auth, ignoring extras), so
  the additive header cannot break the current path-based auth.
- The token is carried in **both** places this cycle: the new `Authorization`
  header and the existing URL path. The path remains the carrier the control
  host actually authenticates on today; the header is added ahead of the
  server-side follow-up that will make it authoritative and let the path token
  be removed.
- **Coordination note (cross-repo, confirmed):** the control host today
  authenticates the agent from the *path* token in its `/ws/agent/<token>/`
  route and does **not** yet read the `Authorization` header. Reading the header
  server-side is a coordinated follow-up in the control-host repo. Therefore
  this change is **additive only**: send the header now **and keep `wsURL()`
  emitting the path token** as the still-authoritative carrier — do not drop the
  path token in this cycle, as that would break auth.
- `wsURL()` keeps the `wss://<host>/ws/agent/<token>/` path form, with a comment
  noting the path token is the current (soon-to-be-deprecated) auth carrier and
  that removing it is deferred until the control host reads the header. The
  header is sent in addition, in preparation for that follow-up.
- The `ws` vs `wss` scheme selection (localhost → `ws`) is preserved.
- A test asserts the dial is invoked with an `Authorization` header whose value
  contains the token.
