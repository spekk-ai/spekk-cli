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

## Bug (fixed): Browser open fails in watch mode

Resolved — `resolveOpenUrl()` now detects HTTP URLs and skips `file://` prefix.

## Bug: Missed reloads during rapid edits

When a file change fires while the browser is mid-reload (old SSE connection closed, new one not yet connected), `notifyClients()` writes to zero clients and the change is silently lost. The browser loads stale HTML and never gets another event until the *next* file change.

**Fix in `src/show/cli.js` `showSpekkWatch()`:** Track a `dirty` flag. When the watcher fires, set `dirty = true` AND call `notifyClients()`. In the server (`src/show/server.js`), when a new SSE client connects, if `dirty` is true, immediately send a `reload` event and reset the flag. This way, if the browser reconnects after missing an event, it gets an immediate reload.

Concretely:
- `startWatchServer` should accept an optional `isDirty` callback and call it when a new SSE client connects. If `isDirty()` returns true, immediately write a `reload` event to that client.
- In `showSpekkWatch()`, maintain `let dirty = false`. Watcher sets `dirty = true` then calls `notifyClients()`. The `GET /` handler resets `dirty = false` (since the HTML was just freshly served). Pass `isDirty: () => dirty` to `startWatchServer`.

## Enhancement: Preserve UI state across live reloads

`location.reload()` resets all interactive UI state. The SSE client script should save and restore state via localStorage around each reload.

**State to preserve:**
1. **Expanded specs** — which spec dropdowns are open (IDs of elements with `.expanded` class under `[id^="assertions-"]`)
2. **Selected detail panel** — which assertion/spec detail is active (ID of the `.detail-content.active` element)
3. **Sidebar scroll position** — `scrollTop` of the sidebar `.spec-list` container

**Implementation (all changes within the SSE client `<script>` block in `src/show/cli.js`):**
- Before `location.reload()`: save expanded spec IDs, active detail ID, and scroll position to `sessionStorage` under a `spekkWatchState` key
- On page load (in watch mode only): read `spekkWatchState` from sessionStorage, re-expand saved specs, re-activate the saved detail panel, restore scroll position, then clear the stored state
- Use `sessionStorage` (not `localStorage`) since this state is ephemeral to the watch session
- Restore scroll position via `requestAnimationFrame` to ensure DOM is rendered first

**What's already persisted (no changes needed):**
- Completed specs toggle — uses `localStorage.getItem('spekkShowCompleted')`
- Metro map height — uses `localStorage.getItem('spekkMetroMapHeight')`
- Metro map collapsed — uses `localStorage.getItem('spekkMetroMapCollapsed')`
