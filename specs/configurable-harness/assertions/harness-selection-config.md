---
id: harness-selection-config
parent: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 2
status: not_started
depends-on: harness-profile-abstraction
branch: feature/configurable-harness
---

# Harness is selected by flag, then env, then default

The active harness resolves from an explicit flag, falling back to an
environment variable, falling back to the built-in default.

## Success Criteria

- Precedence is `--harness` flag > `SPEKK_HARNESS` env var > default
  `claude-code`. A test covers all three levels, including flag overriding env.
- Commands that launch an agent (`spekk coach`, `spekk builder`, and any other
  agent-spawning command) accept a `--harness <name>` flag.
- `--harness` and `SPEKK_HARNESS` accept the canonical names and aliases: an
  unset value and `claude`/`claude-code` all resolve to the claude-code profile;
  `opencode` resolves to the opencode profile.
- An unknown harness name fails fast with an error listing the valid names,
  rather than attempting to spawn a nonexistent binary. The unknown-name error
  is identical whether the name came from the flag or the env var.
