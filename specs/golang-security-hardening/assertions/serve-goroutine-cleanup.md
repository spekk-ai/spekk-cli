---
id: serve-goroutine-cleanup
parent: golang-security-hardening
created: 2026-04-24T12:00:00Z
priority: 2
status: done
depends-on: serve-json-error-handling
branch: feature/golang-migration
---

# WebSocket pipe-reader goroutines use context for cancellation

The stdout and stderr reader goroutines in `internal/serve/serve.go` are tied to a cancellation context. When the WebSocket connection closes or the Claude process exits, the context is cancelled, unblocking any goroutines that might otherwise hang on a broken pipe.

## Success Criteria

- A `context.WithCancel` is created per WebSocket connection
- The cancel function is called on connection cleanup (both normal close and error paths)
- Pipe-reader goroutines check `ctx.Done()` alongside their scanner loops
- The Claude process wait goroutine calls cancel after the process exits
