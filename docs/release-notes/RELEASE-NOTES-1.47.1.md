# Spekk CLI 1.47.1 - Clear Repository Clone Errors and Readable Agent Settings

This patch release fixes repository preparation on sandboxes whose agent settings were written by `setup-credentials.sh`. It also shows the cause of a failed Git step so an operator can tell why repository preparation failed. There are no new commands or flags.

## Repository failures show the Git cause

When a Git command fails during repository preparation, the turn failure reason and agent log now include the last five non-empty lines of Git stderr after the exit status, up to 600 bytes. The agent replaces credentials in URL user information, recognized GitHub token formats, and authorization headers with `<redacted>`. A failed clone can now show a Git cause instead of only `git clone failed: exit status 128`.

The Git credential helper also includes the read error when it cannot open `/etc/spekk/agent.env`. At start, the agent logs one warning with the path and read error if it cannot open that file. It continues to start, but repository chat will fail until the file is readable.

## Agent settings file ownership

`infrastructure/sandbox/setup-credentials.sh` now gives `/etc/spekk/agent.env` to `agent:agent` and sets mode 600 when it writes the file. The service and Git credential helper run as `agent` and need to read it.

**Existing hosts need a separate correction.** A host with a root-owned `/etc/spekk/agent.env` gets the owner fix only when `setup-credentials.sh` runs again, or when an operator runs `sudo chown agent:agent /etc/spekk/agent.env` on that host. Installing the new binaries alone does not change the file owner.
