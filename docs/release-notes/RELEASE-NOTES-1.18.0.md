# Spekk CLI 1.18.0 — The Agent Token Leaves the WebSocket URL

The sandbox agent-client used to send its auth token in both the WebSocket URL path and an `Authorization: Bearer` header. The control host now authenticates on the header alone, which made the path token pure liability: it was the last place the token appeared in a URL, where it can leak into proxy logs, server access logs, and dial-error strings.

The client now dials the path-less route and the header is the sole carrier of the token. A dial failure that echoes the target URL can no longer surface it.

## Upgrade

```bash
spekk update
```
