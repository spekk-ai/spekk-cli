---
id: serve-json-error-handling
parent: golang-security-hardening
created: 2026-04-24T12:00:00Z
priority: 1
status: done
depends-on: websocket-origin-validation
branch: feature/golang-migration
---

# Serve WebSocket handler logs JSON unmarshal errors

All `json.Unmarshal` calls in the WebSocket message handler (`internal/serve/serve.go` switch cases for chat, element_selection, action_recording, init) check and log errors instead of silently discarding them. Malformed messages produce a debug log and are skipped without crashing.

## Success Criteria

- Each `json.Unmarshal` in the incoming-message switch checks its error return
- On unmarshal failure, a debug-level log is emitted with the event type and error
- The malformed message is skipped (no `sendToClaude` call)
- `json.Marshal` error in `sendToClaude` is also checked and logged
- `stdinPipe.Write` error is checked and logged
