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
- The supported harnesses are exactly: `claude-code` (alias `claude`),
  `opencode`, `hermes`, `codex`, and `gemini`. `aider` is not supported and
  resolves as an unknown harness.
- No `aider` profile is registered; the string `aider` appears in no harness
  profile, alias, or valid-name list.
- An unknown harness name fails fast with an error listing the valid names,
  rather than attempting to spawn a nonexistent binary. The unknown-name error
  is identical whether the name came from the flag or the env var.
- The valid-names list shown in that error names each harness once: aliases are
  not listed as separate bare entries. `claude` appears as an alias of
  `claude-code` (e.g. `claude-code (alias: claude)`), not as a standalone name
  that reads like a distinct harness.
