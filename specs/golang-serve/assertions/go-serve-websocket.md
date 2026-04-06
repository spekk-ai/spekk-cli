---
id: go-serve-websocket
parent: golang-serve
created: 2026-04-05T12:25:00Z
priority: 2
status: done
depends-on: go-prompt-resolver
branch: feature/golang-serve
---

# Go serve command runs WebSocket server

The `spekk serve` command in Go starts a WebSocket server that bridges browser extension messages to Claude Code.

## Success Criteria

- `spekk serve` starts WebSocket server on default port 3118
- `spekk serve --port <n>` uses custom port
- `spekk serve --host <host>` uses custom host
- WebSocket accepts connections and handles message types: chat, element_selection, action_recording
- On chat message: spawns claude process with coach prompt + message content
- Streams claude output back to WebSocket client as response messages
- Formats messages correctly (chat → claude input, claude output → chat response)
- Multiple concurrent sessions supported
- SIGINT shuts down server and all claude processes gracefully
- Health check endpoint at `/health`
