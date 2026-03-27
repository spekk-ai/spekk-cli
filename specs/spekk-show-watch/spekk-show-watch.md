---
id: spekk-show-watch
created: 2026-02-26T18:00:00Z
priority: 1
---

# Live-Reload Watch Mode for `spekk show`

`spekk show --watch` starts a local HTTP server that serves the spec explorer and live-reloads the browser via Server-Sent Events (SSE) whenever spec files change.

## Architecture

The current `spekk show` generates a static HTML file and opens it via `file://`. Watch mode adds:

1. **File watcher** (`src/show/watcher.js`) - Watches `specs/` directory for changes, debounces rapid edits
2. **HTTP server with SSE** (`src/show/server.js`) - Serves the generated HTML over HTTP and pushes reload events via SSE `/events` endpoint
3. **SSE client script** - Injected into generated HTML when in watch mode; listens for reload events and refreshes the page
4. **CLI integration** - `--watch` flag parsed via existing `parseFlags()` utility, `showSpekk()` branches between static-file mode and watch-server mode

## Design Decisions

- **No new dependencies** - Uses only Node.js stdlib (`http`, `fs.watch`)
- **SSE over WebSocket** - One-direction push is all we need; SSE is simpler, no handshake, native browser support
- **Debounced regeneration** - File watcher debounces to ~300ms to avoid thrashing during batch edits
- **Graceful shutdown** - Ctrl+C cleanly closes server, watcher, and all SSE connections
