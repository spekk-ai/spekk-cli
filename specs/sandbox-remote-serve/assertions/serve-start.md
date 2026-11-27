---
id: serve-start
parent: sandbox-remote-serve
created: 2026-11-08T00:00:00Z
priority: 2
status: not_started
branch: feature/sandbox-remote-serve
---

# `spekk sandbox serve start` leaves serve running as `agent` with the agent environment

## Success Criteria

- `spekk sandbox serve start <name>` leaves `spekk serve` running on the sandbox as the `agent` user, with every variable from `/etc/spekk/agent.env` in its environment. The `claude` it spawns runs as `agent` too.
- The service keeps running after the SSH session that started it ends, and its output goes to the system journal, not to a file in the login user's home.
- Serve listens on localhost only, on a port the OS picks. Nothing listens on a public interface.
- The nonce and the actual port are written to a file on the sandbox that only `agent` and root can read (mode 600). The file is removed when serve stops.
- Serve's working directory is a directory of its own, never one the worker prepares for a turn.
- Running `start` again while serve is up starts nothing new and reports the running instance. If `agent.env` changed after serve started, `start` restarts it so it picks up the new values, and says so.
- By default the output shows the sandbox, the port, and the next command (`spekk sandbox serve connect <name>`), with no nonce. `--json` prints one object with `nonce`, `port`, and `started_at`.
- `start` refuses with a clear message, and starts nothing, when the sandbox's `spekk` has no `serve` command, when the sandbox's Claude Code fails the same version check the worker applies, or when the login user cannot run privileged commands.
- The command is run against a sandbox name in `~/.config/spekk/sandboxes.json`. An unknown name fails with the same error as `spekk sandbox ssh`.

**Note:** serve must never run with `-v`, which logs message content.
