---
id: watch-server-module-exists
parent: spekk-show-watch
created: 2026-02-26T18:00:00Z
priority: 1
status: not_started
branch: feature/spekk-show-watch
---

# Watch server module exists at `src/show/server.js`

An HTTP server that serves the spec explorer HTML and pushes live-reload events via SSE.

## Success Criteria

- `src/show/server.js` exports a `startWatchServer({ getHTML, port })` function
- Server serves the HTML returned by `getHTML()` at `GET /`
- Server has an SSE endpoint at `GET /events` that keeps connections open
- Server exposes a `notifyClients()` method that sends a `reload` event to all connected SSE clients
- When an SSE client receives `reload`, the browser-side script calls `location.reload()`
- `getHTML()` is called fresh on every `GET /` request (so regenerated HTML is always served)
- Server binds to `localhost` only (not `0.0.0.0`) for security
- Default port is `3117`; if port is in use, tries next port up to 10 times
- Returns `{ server, port, notifyClients, close }` for the caller to orchestrate
- Sets `Content-Type: text/html` for `/` and `text/event-stream` for `/events`
- SSE responses include `Cache-Control: no-cache` and `Connection: keep-alive` headers
- Test file at `src/__tests__/watch-server-module.test.js` validates HTTP serving and SSE behavior

## Interface Contract

```js
// src/show/server.js
export async function startWatchServer({ getHTML, port = 3117 }) {
  // starts HTTP server on localhost:port
  // GET / -> serves getHTML() result as text/html
  // GET /events -> SSE stream
  // returns { server, port, notifyClients: () => void, close: () => void }
}
```
