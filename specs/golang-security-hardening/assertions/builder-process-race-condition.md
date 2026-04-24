---
id: builder-process-race-condition
parent: golang-security-hardening
created: 2026-04-24T12:00:00Z
priority: 1
status: done
branch: feature/golang-migration
---

# Builder agent protects activeProcess with synchronization

The `activeProcess` pointer in `internal/agent/builder.go` is accessed from both the main goroutine (which sets it during `launchClaude`) and a signal-handler goroutine (which reads it to forward SIGINT). Access is synchronized with a mutex to prevent data races.

## Success Criteria

- `activeProcess` reads and writes are protected by a `sync.Mutex`
- The signal-handler goroutine locks the mutex before reading `activeProcess`
- `launchClaude` locks the mutex before writing `activeProcess`
- `go test -race ./internal/agent/...` passes without data race warnings
