---
id: watch-error-page-reloads
parent: show-command
created: 2026-10-17T00:00:00Z
priority: 1
status: done
---

# The `spekk show --watch` error page reloads when the specs parse again

## Description

When a save makes the specs fail to parse (for example a `depends-on` that names no assertion), `spekk show --watch` serves an error page. Before this fix, the error page was plain text with no live-reload script, so it did not reload when the user fixed the file (issue #273).

## Success Criteria

- The error page is HTML. It shows the error text, escaped, and it has the same live-reload script as the explorer.
- The HTTP status of the error page stays 500.
- Live check: in a headless browser, open the explorer, save a `depends-on` that names no assertion, and see the error page. Fix the file and see the explorer come back with no manual refresh.
