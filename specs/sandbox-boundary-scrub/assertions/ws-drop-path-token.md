---
id: ws-drop-path-token
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 2
status: not_started
---

# Agent Auth Token Leaves the WebSocket URL Path

**Blocked on (cross-repo):** the control host must first authenticate the agent
from the `Authorization` header (sent since `ws-auth-header`) instead of the
`/ws/agent/<token>/` path segment. That server-side work is in flight in the
private control-host repo. spekk has no cross-repo `depends-on`, so this
assertion is held at **draft** until the control host ships header auth; flip it
to `not_started` (and build it) once that server change is deployed.

`ws-auth-header` (done) made the token travel in **both** the URL path and an
`Authorization: Bearer` header, additively, because the control host still
authenticates on the path token today. Once the server authenticates on the
header, the path token is pure liability: it is the last place the token appears
in the URL, where it can leak into logs, proxies, and error strings. This
assertion removes it.

## Success Criteria

- `cmd/sandbox/client.go`'s `wsURL()` dials the path-less route
  `wss://<host>/ws/agent/` — the `<token>` segment is gone. The `ws` vs `wss`
  scheme selection (localhost → `ws`) is preserved.
- The `Authorization: Bearer <token>` header built in `dialOptions()` is the
  **sole** carrier of the token; no code path puts the token in the URL.
- The now-stale compatibility comment in `wsURL()` (token sent in both the
  path and the header) is removed or updated.
- `TestDialOptionsSendsAuthorizationHeader` still passes, and a test asserts the
  URL returned by `wsURL()` does **not** contain the token.
- **Dial-failure log no longer leaks the token.** With the token out of the
  path, the reconnect log in `Run()` —
  `log.Printf("Connection lost: %v. ...", err)`, where `err` wraps the
  `websocket.Dial` failure and that failure can echo the target URL — can no
  longer surface the token. Confirm (via test or reasoned check) that a dial
  error routed through this log path contains no token once `wsURL()` is
  path-less.
