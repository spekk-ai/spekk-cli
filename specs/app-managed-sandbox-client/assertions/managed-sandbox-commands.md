---
id: managed-sandbox-commands
parent: app-managed-sandbox-client
created: 2026-10-22T00:00:00Z
priority: 3
status: not_started
branch: feature/app-managed-sandbox-client
depends-on: managed-app-client
---

# Commands launch, send, read, and clean up

## Success criteria

- A distinct command family such as `spekk app sandbox` offers `options`, `repositories`, `launch`, `list`, `status`, `wait`, `cancel`, `destroy`, `send`, and `result`. These names are proposed until implemented. `options` shows only permitted organizations and server launch choices; `repositories` searches the selected organization's full authorized set. `launch` requires `--organization` and a team or personal audience, size, region, kind, and optional repository. It uses the app API and never invokes `spekk sandbox create` or a provider. A team launch remains admin-only and uses the organization's model credential; a personal launch reports the remote sign-in step and does not claim unattended readiness.
- Each mutating command generates a request ID before network I/O, writes its local operation record atomically under user-only permissions, and includes that ID in the request. On timeout, it resolves the same ID through the app before deciding whether to retry. It never silently starts a second paid sandbox or dispatches a task twice. If the app cannot resolve the ID, the command reports an uncertain outcome and stops automatic retries. Client exit has no remote cancel or destroy effect.
- `wait` has a bounded timeout and prints the current launch stage, failure stage, error, agent ID, and expiry. `send` returns the stable launch, agent, request, and message IDs promptly for long work. `result` reads the saved state and reply by message ID, can poll within a bounded timeout, and can continue with the returned Claude session ID only under server `may_resume`. Streaming is optional; after a disconnect the client reads saved status with the same message ID and does not send another POST. Two parallel tasks have separate request, message, and session IDs.
- A machine-readable mode writes one JSON object per result with stable fields: `request_id`, `launch_id`, `agent_id`, `message_id`, `session_id`, `state`, `reply`, and `error` where applicable. Human output gives the same IDs and the next command. No output includes a secret. `cancel` applies to a launch before online; `destroy` applies to an online sandbox. The user chooses retain or destroy explicitly. The app's expiry and cleanup rules continue after local exit.
- A cold terminal run against a local app with a fake cloud covers launch, readiness, text send, saved result, continuation, stream loss, client restart, denied member team launch, denied repository push, offline agent, quota refusal, revoked key, and cleanup. It checks the app records for one launch and one message per request ID.
