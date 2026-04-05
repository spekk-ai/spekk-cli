---
id: go-show-watch-mode
parent: golang-show
created: 2026-04-05T12:23:00Z
priority: 2
status: not_started
depends-on: go-show-html-generation
branch: feature/golang-migration
---

# Go show watch mode with live reload

The `spekk show --watch` mode watches for spec file changes and pushes updates to the browser via SSE.

## Success Criteria

- `spekk show --watch` / `spekk show -w` starts file watcher and HTTP server
- Watches `specs/` directory for `.md` file changes
- On change: re-parses specs, regenerates HTML, sends SSE event to connected browsers
- Browser receives SSE event and reloads while preserving UI state (expanded specs, active detail panel, scroll position)
- HTTP server binds to `127.0.0.1` (IPv4) on an available port
- Server URL printed to console
- SIGINT shuts down watcher and server gracefully
