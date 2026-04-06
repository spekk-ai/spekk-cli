---
id: go-status-display
parent: golang-status
created: 2026-04-05T12:20:00Z
priority: 2
status: done
depends-on: go-parser-json-matches-node
branch: feature/golang-parser
---

# Go status command displays spec overview

The `spekk status` command in Go displays a formatted overview of all specs, assertions, and progress.

## Success Criteria

- Displays each spec with status icon, title, and completion ratio (done/total)
- Assertions listed under their parent spec with indentation, sorted by priority then created
- Status icons: done=check, in_progress=construction, not_started=clipboard, draft=memo
- Overall statistics: total assertions, done, in progress, not started, blocked counts
- Completion percentage displayed
- Next priority item shown with title, parent spec, priority, status, and file path
- Empty specs directory shows helpful getting-started message
- Handles all spec scenarios: complete, partial, empty
