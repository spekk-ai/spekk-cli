---
id: cancel-stops-the-session
created: 2026-10-23T05:00:00Z
priority: 1
---

# A cancel stops the session, not only its running process

## Problem

The control host sends a `cancel` frame to stop an agent session. `WorkerPool.Cancel` (`cmd/sandbox/worker_pool.go`) only signals the `claude` process that the session's worker runs at that moment. This has two gaps (issue #212):

- **The queue continues.** A session can have follow-up messages in `w.msgs`. After the running turn takes SIGTERM, the runner takes the next message and starts a new turn. The host sees turns that run after it cancelled the session.
- **An early cancel is lost.** `Dispatch` registers the worker before `invoke` sets `w.current`. Between the two, `invoke` prepares the repository, the spool, and the attachments, and then starts the process. A cancel in that window finds no process, does nothing, and the turn runs to the end.

The control host opens one turn for each message that it dispatches, and it closes the oldest open turn of a session when a terminal frame (`result` or `error`) for that session arrives. A message that the client drops with no frame leaves its turn open on the host.

## Solution

- A cancel removes every queued message of the session, and the client sends one terminal `error` frame for each removed message, so that the host closes each turn.
- A cancel that arrives before the turn's process starts is recorded on the worker, and the turn does not start its process. The turn ends with a terminal `error` frame.
- A message that arrives after the cancel is a new request and runs as usual.

## What this does not do

- **No protocol change.** The frames are the existing `error` frame with `agent_session_id`. `ProtocolVersion` does not change.
- **No cancel of the repository preparation or of the attachment downloads that are in progress.** They finish, and then the turn stops before its process starts. To stop them in the middle needs a cancel context through the git and download code, which is a larger change for a short window.
- **No new lifetime machinery.** No per-turn context and no cancel registry. One flag on the worker, under the worker mutex, is the full change.
