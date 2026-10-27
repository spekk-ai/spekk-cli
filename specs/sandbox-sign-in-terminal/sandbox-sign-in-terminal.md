---
id: sandbox-sign-in-terminal
created: 2026-10-18T00:00:00Z
priority: 1
---

# The Owner of a Sandbox Signs In to Claude Code Through a Terminal That the Agent Relays

## Problem

A sandbox with model credential `none` (`specs/sandbox-machine-mode/`) has no Claude credential. Its owner must sign in to their own Claude account on the box, with the login of the unmodified Claude Code binary, so that the sign-in completes through Anthropic's own flow. Today the only way is SSH to the box as the agent user, and the owner of a sandbox that a web app launched has no SSH key. The heartbeat already says `claude_signed_in: false`, but nothing lets the person fix it from the app.

## Solution

The agent client accepts four new frames from the control host on its WebSocket, and runs one program in a pseudo-terminal: `claude auth login`, with the `claude` on its `PATH` (the binary that `spekk sandbox deploy` installs and checks against the minimum version). The control host relays the bytes between the terminal and a browser.

- **Frames in.** `terminal_open` (`terminal_id`, `cols`, `rows`, `agent_id`, `user_id`), `terminal_input` (`terminal_id`, `data`), `terminal_resize` (`terminal_id`, `cols`, `rows`), `terminal_close` (`terminal_id`). `data` is base64, because a read of a terminal can end in the middle of a UTF-8 character.
- **Frames out.** `terminal_output` (`terminal_id`, `data`, base64) and `terminal_exit` (`terminal_id`, `exit_code`).
- **No command field.** No frame names a program, an argument, or an environment variable. The agent runs `claude auth login` and nothing else.
- **No record.** The agent writes no `data` to a log or a file. It writes one log line when a terminal opens and one when it closes.
- **The sign-in check reads no credential.** `claude_signed_in` is true when a model credential variable is set, or when the Claude Code sign-in file exists, the agent can open it, and its size is more than zero. The agent never reads the content of the file.

The new frames are additive, so `ProtocolVersion` goes from `1.2` to `1.3`. A control host that does not know them never sends them.

### Rejected

- **A general shell.** The sign-in needs only the Claude Code login. A shell is a larger feature with its own risks.
- **`claude setup-token` and a token that the person pastes into the app.** The app would collect a Claude credential.
- **More than one terminal at a time.** One login at a time is enough. A second `terminal_open` ends the first terminal, so that a terminal that a browser left behind does not block a new one.
- **A text `data` field.** A UTF-8 character that a read cuts in two arrives as a replacement character in JSON text.

## Assertions

1. [agent-sign-in-terminal](assertions/agent-sign-in-terminal.md) - The agent runs only the Claude Code login in a pseudo-terminal, relays its bytes, and records none of them.
2. [sign-in-check-reads-no-credential](assertions/sign-in-check-reads-no-credential.md) - The sign-in check reads no credential, and the heartbeat after a login is sent at once.
