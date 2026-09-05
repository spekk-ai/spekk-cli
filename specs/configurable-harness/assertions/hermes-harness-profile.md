---
id: hermes-harness-profile
parent: configurable-harness
created: 2026-09-05T00:00:00Z
priority: 2
status: done
depends-on: harness-flags-verified-against-cli
branch: feature/configurable-harness
---

# A hermes profile drives coach/builder/observer through the Hermes Agent CLI

Selecting the hermes harness runs the agents through the Hermes Agent CLI, whose
flags differ from both claude and opencode.

## Success Criteria

- `--harness hermes` / `SPEKK_HARNESS=hermes` resolves to a hermes profile.
- Every flag and subcommand the hermes profile emits appears in the installed
  hermes `--help` (top-level and any subcommand used). Confirmed reference points
  from the real CLI: `-z <prompt>` seeds a prompt, `--yolo` auto-approves
  (skip-permissions equivalent), `--cli`/`--tui` select non-interactive vs
  interactive mode, and `chat` is the interactive subcommand.
- Interactive mode launches hermes so it carries the agent prompt and waits for
  input; headless mode runs a single message non-interactively.
- The profile's permission-skip flag is hermes's real one (`--yolo`), not a
  copied claude/opencode flag.
- `observer_cron.go` bakes the hermes binary into the crontab entry.
- The not-found error names Hermes and links its install docs.
- A test asserts the resolved argv for hermes in interactive and headless modes.

**Tests:** internal/agent/harness_test.go
