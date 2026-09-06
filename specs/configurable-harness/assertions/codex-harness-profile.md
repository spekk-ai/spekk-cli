---
id: codex-harness-profile
parent: configurable-harness
created: 2026-09-05T00:00:00Z
priority: 3
status: not_started
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# A codex profile drives coach/builder/observer through the codex CLI

Selecting the codex harness runs the agents through the codex CLI. codex is not
installed in the working environment, so the builder must install it (or run
where it is available) and read its real `--help` before this can be done — no
flags from memory.

## Success Criteria

- `--harness codex` / `SPEKK_HARNESS=codex` resolves to a codex profile.
- Every flag and subcommand the codex profile emits is confirmed present in a
  real `codex --help` — no flag is taken from memory or another harness.
- Interactive mode opens a skill-governed codex session that waits for input
  (per `interactive-uses-installed-skill`) — the full agent prompt is not passed
  as an executed message. Headless mode runs a single message non-interactively;
  the profile uses codex's real permission-skip flag.
- `observer_cron.go` bakes the codex binary into the crontab entry.
- The not-found error names codex and links its install docs.
- A test asserts the resolved argv for codex in interactive and headless modes.
- This is not done until the flags have been verified against the installed
  codex binary (per `harness-flags-verified-against-cli`); an absent binary
  means the assertion stays open, not fabricated.

**Tests:** internal/agent/harness_test.go, internal/agent/harness_cliverify_test.go
