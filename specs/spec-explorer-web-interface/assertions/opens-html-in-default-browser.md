---
id: opens-html-in-default-browser
parent: spec-explorer-web-interface
created: 2026-01-22T21:00:00Z
priority: 3
status: done
---

# Opens HTML in Default Browser

## Assertion

The command automatically opens the generated HTML file in the system's default browser after generation.

## Success Criteria

- After running `spekk show`, the default browser launches automatically
- Browser opens to the correct local file path (`.spekk/index.html`)
- Works across different operating systems (macOS, Linux, Windows)
- Command completes successfully even if browser fails to open
- No browser windows open if generation fails

## Test Plan

**Tests:** src/__tests__/show-command.test.js

```bash
spekk show
# Verify browser opens with correct file
# Test on different platforms if possible
```