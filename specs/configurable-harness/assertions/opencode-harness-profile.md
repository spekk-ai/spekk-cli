---
id: opencode-harness-profile
parent: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 2
status: done
depends-on: harness-profile-abstraction
branch: feature/configurable-harness
---

# An opencode profile drives the interactive, headless, and observer launches

Selecting the opencode harness runs the coach, builder, and observer through the
opencode CLI instead of `claude`, with opencode's own flag conventions.

## Success Criteria

- An opencode profile exists whose binary and flags match the opencode CLI's
  actual conventions for: passing a prompt, running non-interactively/headless,
  and skipping permission prompts. (The builder confirms the real opencode flag
  names; the profile is not a copy of the claude flags.)
- With `SPEKK_HARNESS=opencode` (or `--harness opencode`), the interactive and
  headless launch sites spawn the opencode binary, and `observer_cron.go` bakes
  the opencode binary into the crontab entry.
- The opencode profile's not-found error names opencode and points to opencode's
  install instructions, not Claude's.
- A test asserts the resolved argv for the opencode profile in both interactive
  and headless modes.
- Selecting opencode changes only the launch harness; the prompts, skills, and
  spec workflow are unchanged.

**Tests:** internal/agent/harness_test.go
