---
id: serve-app-handoff
parent: sandbox-remote-serve
created: 2026-11-08T00:00:00Z
priority: 3
status: not_started
depends-on: serve-connect
branch: feature/sandbox-remote-serve
---

# A remote coach session can continue in the app

When the sandbox is registered with the control host, a developer can hand the current coach session to the app, then continue it there with full context.

## Success Criteria

- In a `connect` session, `/handoff [title]` asks the worker on the same sandbox to send a `conversation_open` frame. The frame carries the coach's Claude session ID, a title (default: `Coach session on <name>`), and a body that summarizes the session so far. The worker stamps the session ID; the client does not supply it in the frame.
- The worker accepts a handoff only for a Claude session that serve started on that machine. It refuses any other session ID, including the worker's own, and logs the refusal.
- When someone replies in the app conversation, the worker resumes the coach session with `--resume` in serve's working directory, so the reply has the full coach context.
- After a successful handoff, `connect` prints that the session continues in the app and exits. The control host's typed rejections (`conversation_open_*`) print as one clear line, and the local session stays open.
- On a sandbox with no agent ID, or with the worker not connected, `/handoff` fails with a message that says why. Every other `connect` behavior is unchanged.
- `connect` lists `/handoff` in its startup hint only when the sandbox has an agent ID.
