---
id: golang-serve
created: 2026-04-05T12:25:00Z
priority: 2
---

# Go Serve Command

Port the `spekk serve` WebSocket server from Node.js to Go.

The serve module runs a WebSocket server that bridges a browser extension to Claude Code for interactive spec coaching. Uses the `ws` npm package currently.

## Current Architecture

- `src/serve/index.js` — WebSocket server, claude process management (312 lines)
- `src/serve/message-formatter.js` — Format messages between browser and claude (266 lines)
- `src/serve/shapes.js` — TN Models WebSocket adapter shapes (131 lines)
- `src/serve/cli.js` — CLI entry point (43 lines)

## Strategy

- Use gorilla/websocket or nhooyr.io/websocket for WebSocket server
- Port message formatting to Go
- Claude subprocess management same pattern as agent launchers
