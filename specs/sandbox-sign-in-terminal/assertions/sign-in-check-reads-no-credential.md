---
id: sign-in-check-reads-no-credential
parent: sandbox-sign-in-terminal
created: 2026-10-18T00:00:00Z
priority: 1
status: not_started
branch: feature/personal-sign-in-terminal
depends-on: agent-sign-in-terminal
---

# The Sign-In Check Reads No Credential, and the Heartbeat After a Login Is Sent at Once

## Success Criteria

- `claudeSignedIn` is true when one of the credential variables of `agent-reports-sign-in-required` is set, or when the Claude Code sign-in file (`$CLAUDE_CONFIG_DIR/.credentials.json`, else `$HOME/.claude/.credentials.json`) is a regular file that the agent can open and whose size is more than zero. The agent opens the file and closes it with no read. It never reads the content. This replaces the rule "not empty after the white space is removed" of `agent-reports-sign-in-required`.
- When a sign-in terminal ends, the agent sends a heartbeat frame (with `claude_signed_in`) before the `terminal_exit` frame, so that the control host knows the new state when the browser hears of the exit.

## Verification

- The table test of the check: no credential, one variable, the file under `HOME` and under `CLAUDE_CONFIG_DIR`, an empty file, a file that the agent cannot open.
- The terminal test checks that a heartbeat frame comes before `terminal_exit`.
