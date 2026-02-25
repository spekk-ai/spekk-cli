---
id: no-browser-in-tests
parent: spec-explorer-web-interface
created: 2026-02-25T01:00:00Z
priority: 1
status: done
---

# Tests Never Open the Browser

Running `npm test` never triggers the system browser to open. This is a side effect that steals focus and disrupts developer workflow.

## Success Criteria

- `showSpekk()` does not call `openInBrowser()` when `NODE_ENV=test` or `CI=true`
- All test files that exercise HTML generation import `generateSpecExplorerHTML` directly instead of calling `showSpekk()`
- `npm test` can run without any browser windows opening
- The "opens browser" test in `show-command.test.js` validates the mechanism exists (e.g., `openInBrowser` is a function) without actually spawning a browser process
- No other tests call `spekk show` via `execSync` without suppressing browser opening
