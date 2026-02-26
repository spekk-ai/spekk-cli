---
id: watch-mode-integrated
parent: spekk-show-watch
created: 2026-02-26T18:00:00Z
priority: 1
status: done
depends-on: file-watcher-module-exists
branch: feature/spekk-show-watch
---

# Watch mode is integrated end-to-end

The `--watch` flag is parsed, all modules are wired together, and `spekk show --watch` works end-to-end.

## Success Criteria

- `bin/spekk.js` parses `--watch` / `-w` flags for the `show` command using `parseFlags()` from `src/cli/parse-flags.js`
- `showSpekk()` in `src/show/cli.js` accepts an options object `{ watch }`
- When `watch` is false/absent: behavior is unchanged (generate file, open browser, exit)
- When `watch` is true:
  1. Parses specs and generates HTML (with SSE client script injected)
  2. Starts the watch server from `src/show/server.js`, serving a `getHTML` callback that re-parses and re-generates on each request
  3. Starts the file watcher from `src/show/watcher.js` on the `specs/` directory
  4. On file change: calls `notifyClients()` to push SSE reload event
  5. Opens `http://localhost:{port}` in the default browser
  6. Logs: "Watching specs/ for changes... (press Ctrl+C to stop)"
  7. Ctrl+C triggers graceful shutdown: stops watcher, closes server and SSE connections, exits
- Generated HTML in watch mode includes an inline `<script>` with EventSource SSE client that:
  - Connects to `/events`
  - On `reload` event, calls `location.reload()`
  - Auto-reconnects on connection loss (EventSource default behavior)
- Non-watch mode HTML does NOT include the SSE script (no dead code in static output)
- Existing tests in `src/__tests__/show-command.test.js` continue to pass (non-watch path unchanged)
- Integration test at `src/__tests__/watch-mode-integration.test.js` validates the full flow:
  - Server starts and serves HTML
  - Modifying a spec file triggers browser reload event
  - Server shuts down cleanly

## Depends On

- `file-watcher-module-exists` - provides `watchSpecs()`
- `watch-server-module-exists` - provides `startWatchServer()`

Note: only one `depends-on` in frontmatter (the watcher). The server is also required but the coordinator chains them appropriately.

## Bug: Browser open fails in watch mode

`openInBrowser()` in `src/show/cli.js` unconditionally wraps its argument with `file://` prefix (line 89). The non-watch path passes a file path so this is correct. But the watch path passes `http://localhost:{port}` — resulting in `file://http://localhost:3117` which is invalid.

**Fix:** `openInBrowser()` should detect when the input is already a URL (starts with `http://` or `https://`) and pass it directly to the OS open command without the `file://` prefix.
