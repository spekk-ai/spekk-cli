---
id: aider-harness-profile
parent: configurable-harness
created: 2026-09-05T00:00:00Z
priority: 2
status: done
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# An aider profile drives coach/builder/observer through the aider CLI

Selecting the aider harness runs the agents through aider, whose flags differ
again from the other harnesses.

## Success Criteria

- `--harness aider` / `SPEKK_HARNESS=aider` resolves to an aider profile.
- Every flag the aider profile emits appears in the installed `aider --help`.
  Confirmed reference points from the real CLI: bare `aider` is interactive,
  `--message <msg>`/`-m` sends one message then exits (headless), and
  `--yes-always` is the confirmation-skip flag.
- Interactive mode launches bare aider and waits for input; headless mode uses
  `--message` so aider processes a single message and exits.
- The profile's permission-skip flag is aider's real one (`--yes-always`).
- `observer_cron.go` bakes the aider binary into the crontab entry.
- The not-found error names aider and links its install docs.
- A test asserts the resolved argv for aider in interactive and headless modes.

**Tests:** internal/agent/harness_test.go, internal/agent/harness_cliverify_test.go
