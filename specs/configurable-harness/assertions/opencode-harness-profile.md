---
id: opencode-harness-profile
parent: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 2
status: not_started
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# The opencode profile drives coach/builder/observer with opencode's real flags

Selecting the opencode harness runs the coach, builder, and observer through the
opencode CLI. The bare `opencode` command starts the persistent TUI and accepts
`--prompt`, `--agent`, `--model`, and `--auto` (verified against the installed
binary). Interactive must seed the persistent TUI, not `run`: `run` executes its
message and exits, and seeding the full agent prompt auto-runs it as a task.

## Success Criteria

- Every flag and subcommand the opencode profile emits appears in the installed
  binary's `opencode --help` / `opencode run --help`. The bare `opencode` TUI
  defines `--prompt`, `--agent`, `--model`, and `--auto`; `run` is the headless
  subcommand.
- Interactive coach/builder opens the persistent `opencode` TUI seeded with the
  short skill activation via `opencode --prompt <activation>` (per
  `interactive-uses-installed-skill`). It stays open and waits for input — it
  does not use `run -i`, which exits after the message.
- The full agent prompt is never passed via `--prompt`; only the short
  activation is, so opencode does not auto-run the prompt as a build task.
- Headless mode uses `opencode run <message>` with opencode's real
  permission-skip flag (`--auto`).
- `observer_cron.go` bakes the opencode binary into the crontab entry.
- The opencode profile's not-found error names opencode and links opencode's
  install docs, not Claude's.
- A test asserts the resolved argv for opencode in interactive (persistent TUI,
  `--prompt`, no `run`) and headless (`run`) modes, and (per
  `harness-flags-verified-against-cli`) that argv uses only flags the installed
  opencode defines.
- Selecting opencode changes only the launch harness; prompts, skills, and the
  spec workflow are unchanged.
