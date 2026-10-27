---
id: managed-sandbox-commands
parent: app-managed-sandbox-client
created: 2026-10-22T00:00:00Z
priority: 3
status: done
branch: docs/local-agent-cli-contract
depends-on: managed-app-client
---

# Commands launch, send, read, and clean up

## Success criteria

- A distinct command family such as `spekk app sandbox` offers `options`, `repositories`, `launch`, `list`, `status`, `wait`, `cancel`, `destroy`, `send`, and `result`. These names are proposed until implemented. `options` shows only permitted organizations and server launch choices; `repositories` searches the selected organization's full authorized set. `launch` requires `--organization` and a team or personal audience, size, region, kind, and optional repository. It uses the app API and never invokes `spekk sandbox create` or a provider. A team launch remains admin-only and uses the organization's model credential; a personal launch prints the remote sign-in step ("Sign in to Claude" in the agent terminal in the app) in a `sign_in` field and does not claim unattended readiness.
- `launch` and `send` each generate a request ID, a UUID, before network I/O, write a local operation record atomically under user-only permissions, and include the ID in the request. On timeout, the client resolves that ID through the app before it considers a retry. It never automatically starts a second paid sandbox or dispatches a task twice. If the app cannot resolve the ID, the command reports an uncertain outcome and stops automatic retries. A `send --request-id` must be a UUID, as the app chat routes require. When a send gets 409, the client reads the saved status by the request ID, prints it if the app has the request, exits with the app's error, and does not POST again. This covers the app rule that a saved send whose dispatch is not confirmed is never dispatched again. `cancel` and `destroy` use the saved launch ID and the existing launch detail route after a timeout or lost response; they have no request-ID lookup. The client reads the current stage and `finished`, reports an uncertain result when the action is not confirmed, and asks the user to choose any further action. It does not automatically repeat a cleanup POST. Client exit has no remote cancel or destroy effect.
- `wait` has a bounded timeout and prints the current launch stage, failure stage, error, agent ID, and expiry. `send` returns the stable launch, agent, request, and message IDs promptly for long work. `result` reads the saved state and reply by message ID, can poll within a bounded timeout, and can continue with the returned Claude session ID only under server `may_resume`. Streaming is optional; after a disconnect the client reads saved status with the same message ID and does not send another POST. Two parallel tasks have separate request, message, and session IDs.
- A machine-readable mode writes one JSON object per result with stable fields: `request_id`, `launch_id`, `agent_id`, `message_id`, `session_id`, `state`, `reply`, and `error` where applicable. Human output gives the same IDs and the next command. No output includes a secret. `request_id` applies to launch and send only. `cancel` applies to a launch before online; `destroy` applies to an online sandbox. The user chooses retain or destroy explicitly. Cleanup is complete only when stage is `failed` or `destroyed` and `finished` is set. A failed launch with `finished` null can still have a billable machine. The client warns the user while cleanup is incomplete. The app worker and orphan sweep own automatic cleanup, and the user monitors or escalates stalled cleanup. The app's expiry and cleanup rules continue after local exit.
- The cold terminal run against a local app with a fake cloud is the separate assertion `managed-commands-acceptance-run`.

**Tests:** `cmd/spekk/app_test.go`, `internal/appclient/client_test.go`
