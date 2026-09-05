---
id: codex-harness-profile
parent: configurable-harness
created: 2026-09-05T00:00:00Z
priority: 3
status: draft
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# A codex profile drives coach/builder/observer through the codex CLI

Selecting the codex harness runs the agents through the codex CLI. This is
`draft` because codex is not installed in the working environment; it stays out
of the build queue until the binary is present, so no builder fabricates flags
against a missing CLI.

## Success Criteria

- `--harness codex` / `SPEKK_HARNESS=codex` resolves to a codex profile.
- Every flag and subcommand the codex profile emits is confirmed present in a
  real `codex --help` — no flag is taken from memory or another harness.
- Interactive mode carries the agent prompt and waits for input; headless mode
  runs a single message non-interactively; the profile uses codex's real
  permission-skip flag.
- `observer_cron.go` bakes the codex binary into the crontab entry.
- The not-found error names codex and links its install docs.
- A test asserts the resolved argv for codex in interactive and headless modes.
- Promotion out of `draft` requires the flags to have been verified against the
  installed codex binary (per `harness-flags-verified-against-cli`).
