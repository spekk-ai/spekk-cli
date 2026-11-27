---
id: serve-connect
parent: sandbox-remote-serve
created: 2026-11-08T00:00:00Z
priority: 2
status: not_started
depends-on: serve-start
branch: feature/sandbox-remote-serve
---

# `spekk sandbox serve connect` gives a local terminal session with the remote coach

## Success Criteria

- `spekk sandbox serve connect <name>` reads the nonce and port from the sandbox over SSH, opens an SSH tunnel from a free local port, and connects to serve. The user supplies nothing but the name. No WebSocket tool needs to be installed locally.
- In the default mode each line the user types is sent as a `coach:chat` message, and the assistant text from `coach:assistant` prints as it streams. `coach:error` and `coach:agent_exited` print as clear one-line messages.
- `--raw` reads one JSON envelope per line on stdin and writes each received envelope as one JSON line on stdout, for scripts.
- The tunnel uses SSH keepalives, so an idle session survives a quiet network for at least 10 minutes.
- Serve accepts one connection at a time. A second `connect` while one is open fails with a message naming the sandbox, and leaves the first session alone.
- When serve is not running, `connect` exits non-zero within 15 seconds with `serve is not running on <name>; run spekk sandbox serve start <name>`. It does not hang.
- Ctrl-D or Ctrl-C closes the WebSocket and the tunnel. No `ssh` process remains afterwards.
- No output, local file, or process argument list on the local machine contains the nonce.
