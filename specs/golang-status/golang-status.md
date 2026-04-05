---
id: golang-status
created: 2026-04-05T12:20:00Z
priority: 2
---

# Go Status Command

Port the `spekk status` command from Node.js to Go.

Simple module — reads parsed specs and displays a formatted overview to the terminal. 136 lines of JS.

## Strategy

- Reuse the Go parser to get specs + assertions
- Format and print to stdout with status icons and completion stats
