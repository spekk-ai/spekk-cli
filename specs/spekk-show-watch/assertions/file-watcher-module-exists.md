---
id: file-watcher-module-exists
parent: spekk-show-watch
created: 2026-02-26T18:00:00Z
priority: 1
status: done
branch: feature/spekk-show-watch
---

# File watcher module exists at `src/show/watcher.js`

A self-contained module that watches the `specs/` directory for file changes and invokes a callback.

## Success Criteria

- `src/show/watcher.js` exports a `watchSpecs(specsDir, onChange)` function
- `onChange` callback is invoked when `.md` files in `specs/` are created, modified, or deleted
- Rapid successive changes are debounced (~300ms) so `onChange` fires once per batch
- Uses Node.js built-in `fs.watch` with `{ recursive: true }` - no external dependencies
- Returns a cleanup function that stops watching when called
- Test file at `src/__tests__/file-watcher-module.test.js` validates debounce behavior and cleanup

**Tests:** src/__tests__/file-watcher-module.test.js

## Interface Contract

```js
// src/show/watcher.js
export function watchSpecs(specsDir, onChange) {
  // watches specsDir recursively for .md file changes
  // calls onChange() debounced on changes
  // returns () => void cleanup function
}
```
