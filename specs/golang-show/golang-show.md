---
id: golang-show
created: 2026-04-05T12:22:00Z
priority: 2
---

# Go Show Command

Port the `spekk show` web interface generator from Node.js to Go.

The largest single module (~1958 lines). Generates a self-contained HTML file with the spec explorer UI, including metro map visualization, search, detail panels, and optional watch mode with SSE live reload.

## Current Architecture

- `src/show/cli.js` — HTML generation with embedded CSS/JS (1958 lines)
- `src/show/server.js` — SSE server for watch mode (92 lines)
- `src/show/watcher.js` — File watcher using fs.watch (76 lines)

## Strategy

- Go generates the same HTML output using Go templates or string building
- Watch mode uses Go's fsnotify + net/http for SSE
- The HTML/CSS/JS is the product — it's embedded in the Go binary as string constants or embed.FS
