---
id: opencode-harness-profile
parent: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 2
status: in_progress
locked-by: builder-MacBook-Pro.local-7729-1788728856
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# The opencode profile drives coach/builder/observer with opencode's real flags

Selecting the opencode harness runs the coach, builder, and observer through the
opencode CLI. The profile's flags must match the installed opencode binary — an
earlier version emitted flags opencode does not define (`--prompt`, and `--auto`
on the bare command), so the whole prompt was silently dropped and opencode
opened an empty TUI instead of acting as the agent.

## Success Criteria

- Every flag and subcommand the opencode profile emits appears in the output of
  `opencode --help` / `opencode run --help` for the installed binary. In
  particular: opencode has no `--prompt` flag (the message is a bare positional),
  and `--auto` and the message live under the `run` subcommand, not the bare
  `opencode` command.
- Interactive coach/builder opens a skill-governed opencode session that waits
  for input (per `interactive-uses-installed-skill`) — the full agent prompt is
  not passed as an executed message, so opencode does not auto-run it as a build
  task. The bare `opencode` command receives no flags it does not define.
- Headless mode uses `opencode run <message>` with opencode's real
  permission-skip flag.
- `observer_cron.go` bakes the opencode binary into the crontab entry.
- The opencode profile's not-found error names opencode and links opencode's
  install docs, not Claude's.
- A test asserts the resolved argv for opencode in interactive and headless
  modes, and (per `harness-flags-verified-against-cli`) that argv uses only flags
  the installed opencode defines.
- Selecting opencode changes only the launch harness; prompts, skills, and the
  spec workflow are unchanged.

**Tests:** internal/agent/harness_test.go, internal/agent/harness_cliverify_test.go
