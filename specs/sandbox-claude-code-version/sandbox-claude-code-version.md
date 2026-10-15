---
id: sandbox-claude-code-version
created: 2026-10-15T21:30:00Z
priority: 1
---

# A Sandbox Runs Claude Code at or Above a Minimum Version

## Problem

Every provisioning path installed Claude Code with `npm install -g @anthropic-ai/claude-code`. The npm `latest` tag gives 2.1.197, so the sandboxes ran 2.1.197. That version kills a background task about 5 seconds after the `result` event and exits. The agent never sees how the task ended. This is the cause of the failed background-type dispatches that issue #173 measured. A newer version keeps the `claude -p` process alive until the background task ends (or reaches its 600-second limit), starts a new turn, and reports. Opus 5.5 also needs 2.1.280 or newer.

Nothing on a sandbox said which version it ran, and nothing stopped a turn on an old version.

## The minimum is 2.1.292

The minimum is the oldest version that has both behaviors that a sandbox needs. Measured on 2026-10-15 with the native installer, a logged-in `claude -p - --output-format stream-json --verbose --dangerously-skip-permissions`, and a prompt that starts `sleep 15` in the background and ends the turn:

| Version | What happened after the first `result` |
|---|---|
| 2.1.197, 2.1.226 | The task is killed, and the process exits with no second `result`. |
| 2.1.280, 2.1.290, 2.1.291 | The task is killed (`task_updated` `killed`, `task_notification` `stopped`) within seconds, and the process exits with no second `result`. |
| 2.1.292 to 2.1.296, 2.1.301 | The process waits. The task completes, and the agent sends a second `result`. |

2.1.291 and 2.1.292 were each run two times, with the same result. 2.1.301 is also recorded in `cmd/sandbox/testdata/` (see `turn-progress-reporting`). 2.1.280 is enough for Opus 5.5, but it is not enough for background work, so the minimum is 2.1.292.

## Design

- **One place.** The minimum is the constant `MinVersion` in `internal/claudecode`. The sandbox client and `spekk sandbox deploy` both read it. No other code, template, or script holds the minimum.
- **Native installer, as `agent`.** Every provisioning path downloads `https://claude.ai/install.sh` to a file with `curl -fsSL`, and then runs that file with `bash ... stable`, as the `agent` user. A failed download stops the line with the curl error. A pipe (`curl ... | bash`) does not: with no `pipefail`, bash runs an empty script and exits 0. Each path then links `/usr/local/bin/claude` to `/home/agent/.local/bin/claude`. The `spekk-agent` service has the systemd default `PATH`, which has `/usr/local/bin` before `/usr/bin`, so the service finds this binary before an old npm copy in `/usr/bin`. No path installs `@anthropic-ai/claude-code` from npm.
- **The `stable` channel and a minimum, not a pinned version.** The native Claude Code updates itself, so a pinned version does not stay pinned unless the updater is also turned off, and each new pin needs a code change and a deploy to every sandbox. The one fact that the sandbox depends on is a floor, so the code states a floor. `stable` (2.1.292 on 2026-10-15) gets fewer regressions than `latest` (2.1.301), and it is at the minimum. When the minimum moves above `stable`, the deploy fails and says so, and a person decides. The installer argument `stable` also sets the channel that the self-updater follows: `claude install stable` writes `"autoUpdatesChannel": "stable"` to `~agent/.claude/settings.json` (the Claude Code docs: "The channel you choose at install time becomes your default for auto-updates"). The setting has the default `latest` when it is not there. So the install lines set the channel, and no other line writes the setting.
- **Deploy upgrades.** `spekk sandbox deploy` (and the deploy step of `create` and `provision`) reads the version that the service would run, with the service's user and environment file, so a `PATH` in `agent.env` applies to the check too. When it is older than the minimum, or cannot be read, the deploy installs the `stable` release before it restarts the agent. After the install, a version below the minimum stops the deploy, and a version that cannot be read gives a warning, and the deploy continues.
- **The client refuses only a known old version.** Before each turn, the client runs `claude --version` (about 20 ms on the native binary). When it reads a version below the minimum, it does not start the turn, and it sends an `error` frame that a person can act on. When it cannot read a version, it logs a warning and runs the turn.
- **Fail open on an unreadable version.** The native Claude Code updates itself. If a later build prints its version in a new form, a check that refuses on an unreadable version refuses every turn on every sandbox, and a deploy cannot repair it, because the deploy reads the version the same way. So only a version that was read and is below the minimum stops a turn or a deploy. The parse accepts the forms that are likely: `2.1.292 (Claude Code)`, `Claude Code v2.2.0`, and `claude 2.2.0`.

## Rejected

- A configurable minimum (a flag or an environment variable). The minimum is a fact about Claude Code, not a choice per sandbox.
- A version check in the shell scripts. Two comparisons can disagree. The check stays in Go, and the scripts only install.
- Removing the old npm copy during a deploy. The link in `/usr/local/bin` is found first, and the client check catches the case where it is not. Removing a package that something else can use is not needed for this fix.
- A separate line that writes `autoUpdatesChannel` to `settings.json`. The installer writes it from its `stable` argument, and a second writer must merge JSON in a file that the agent also edits.
- A read of `PATH` from `agent.env` in a shell. `agent.env` is a systemd environment file, and its quote and escape rules are not the shell rules. `systemd-run` with the unit's `User=` and `EnvironmentFile=` reads it the way the service does.
- A cached client check. A cache needs a rule for when to read again (after a self-update, or after a failed read). A check at each turn is 20 ms and has no rule.

## The refusal reaches the person

The control host keeps only the `error` field of an `error` frame. It records that text as the reason the turn failed and shows it to the person, and a place that shows it can cut it at 300 characters. It does not read `detail`. So the refusal puts the whole message in `error`, in fewer than 300 characters, and the frame has no other new field. The frame shape is unchanged, so the protocol version does not change.
