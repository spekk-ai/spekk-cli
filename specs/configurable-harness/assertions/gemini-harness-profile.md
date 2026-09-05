---
id: gemini-harness-profile
parent: configurable-harness
created: 2026-09-05T00:00:00Z
priority: 3
status: draft
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# A gemini-cli profile drives coach/builder/observer through the Gemini CLI

Selecting the gemini harness runs the agents through the Gemini CLI. This is
`draft` because the gemini binary is not installed in the working environment;
it stays out of the build queue until the binary is present, so no builder
fabricates flags against a missing CLI.

## Success Criteria

- `--harness gemini` / `SPEKK_HARNESS=gemini` resolves to a gemini profile.
- Every flag and subcommand the gemini profile emits is confirmed present in a
  real `gemini --help` — no flag is taken from memory or another harness.
- Interactive mode carries the agent prompt and waits for input; headless mode
  runs a single message non-interactively; the profile uses the Gemini CLI's
  real permission-skip / non-interactive flag.
- `observer_cron.go` bakes the gemini binary into the crontab entry.
- The not-found error names the Gemini CLI and links its install docs.
- A test asserts the resolved argv for gemini in interactive and headless modes.
- Promotion out of `draft` requires the flags to have been verified against the
  installed gemini binary (per `harness-flags-verified-against-cli`).
