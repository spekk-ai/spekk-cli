---
id: go-orchestration-loops
parent: golang-agents
created: 2026-04-05T12:18:00Z
priority: 2
status: in_progress
depends-on: go-builder-launcher
branch: feature/golang-agents
---

# Go orchestration loops

The Go builder and coach loops continuously run agent sessions with git commit handling between iterations.

## Success Criteria

**Builder loop (`spekk loop builder`):**
- Gets next assertion via parser
- Launches claude builder agent
- After agent completes: stages and commits changes if any
- Handles parser errors (transient: retry after 5s)
- Handles `complete` state (wait 5s, check again)
- SIGINT/SIGTERM exit gracefully
- Colored console output with iteration counter

**Coach loop (`spekk loop coach`):**
- Launches claude coach agent in interactive mode
- After agent completes: stages and commits spec changes if any
- SIGINT/SIGTERM exit gracefully
- Colored console output with session counter
