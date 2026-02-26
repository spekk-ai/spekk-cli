---
id: file-watcher-module-exists
parent: spekk-show-watch
created: 2026-02-26T18:00:00Z
priority: 1
status: in_progress
branch: feature/spekk-show-watch
---

# File watcher module exists at `src/show/watcher.js`

A self-contained module that watches the `specs/` directory for file changes and invokes a callback.

## Success Criteria

- `src/show/watcher.js` exports a `watchSpecs(specsDir, onChange)` function
- `onChange` callback is invoked when `.md` files in `specs/` are created, modified, or deleted
- Rapid successive changes are debounced (~500ms) so `onChange` fires once per batch
- Uses **polling-based approach** (`fs.readdirSync`/`fs.statSync` on a `setInterval`) — not `fs.watch`
- Polls every 500ms, comparing mtimes of all `.md` files against a cached snapshot
- Detects: file modifications (mtime changed), new files added, files deleted
- No external dependencies — Node.js stdlib only
- Returns a cleanup function that clears the interval and stops polling
- Test file at `src/__tests__/file-watcher-module.test.js` validates detection and cleanup

**Tests:** src/__tests__/file-watcher-module.test.js

## Bug: `fs.watch` with `{ recursive: true }` is unreliable on Linux

On Linux, `sed -i` and similar tools replace files via create-temp-then-rename. After the rename, `fs.watch` loses its inotify handles and silently stops firing events. Observed behavior: watcher fires for the first change, then goes silent for all subsequent changes.

**Fix:** Replace `fs.watch` with polling. Scan the directory tree every 500ms, collect mtimes of all `.md` files, diff against previous snapshot. This is reliable across all platforms and file replacement strategies.

## Interface Contract

```js
// src/show/watcher.js
export function watchSpecs(specsDir, onChange) {
  // polls specsDir recursively every 500ms for .md file mtime changes
  // calls onChange() debounced when changes detected
  // returns () => void cleanup function
}
```
