---
id: agent-sign-in-terminal
parent: sandbox-sign-in-terminal
created: 2026-10-18T00:00:00Z
priority: 1
status: done
branch: feature/personal-sign-in-terminal
---

# The Agent Runs Only the Claude Code Login in a Pseudo-Terminal, Relays Its Bytes, and Records None of Them

## Success Criteria

- On `terminal_open`, the agent client starts `claude auth login` (the `claude` that `exec.LookPath` finds) in a pseudo-terminal of `cols` x `rows`, as the user of the agent process, with the environment of the agent process and `TERM=xterm-256color`. The `Message` type has no field that names a program, an argument, or an environment variable.
- Each read of the terminal goes to the control host as `{"type": "terminal_output", "terminal_id": "...", "data": "<base64>"}`. A frame that finds no live connection is dropped.
- `terminal_input` writes the decoded `data` to the terminal. `terminal_resize` sets the size. A frame with a `terminal_id` that is not the open terminal does nothing: input after the exit goes nowhere.
- The terminal ends when the login exits, on `terminal_close`, after 10 minutes, or when a second `terminal_open` arrives (the second one then starts). An end stops the whole process group of the login. The agent then sends `{"type": "terminal_exit", "terminal_id": "...", "exit_code": N}` (`-1` when a signal stopped it).
- The agent log has one line when a terminal opens (the agent id, the user id, the terminal id) and one when it closes (the same, plus the duration and the exit code). No log line holds `data`.
- If `claude` is not on the `PATH`, the agent sends a short `terminal_output` that says so and a `terminal_exit` with code `127`.
- The terminal frames came with `ProtocolVersion` `1.3`. A later additive bump does not change this assertion.

**Tests:** cmd/sandbox/terminal_test.go, cmd/sandbox/claude_auth_test.go

## Verification

- A test opens a terminal with a fake `claude` on `PATH` that prints a URL, reads one line, and exits. It sends a fake code as `terminal_input`, and checks the `terminal_output` frames, the `terminal_exit` code, and that the captured log has neither the URL nor the code.
- The walk checks the 10-minute timeout with a terminal that nobody uses.
- A walk with the real agent binary against the control host, as the person does it in the app.
