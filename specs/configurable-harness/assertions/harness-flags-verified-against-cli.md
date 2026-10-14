---
id: harness-flags-verified-against-cli
parent: configurable-harness
created: 2026-09-05T00:00:00Z
priority: 1
status: done
depends-on: harness-profile-abstraction
branch: feature/configurable-harness
---

# A harness profile only emits flags its binary actually defines

**Tests:** internal/agent/harness_cliverify_test.go

A profile's flags are checked against the real CLI, not written from memory. The
opencode profile shipped flags opencode does not define (`--prompt`, `--auto` on
the bare command); the argv tests passed anyway because they only asserted the
profile's own output. This assertion adds the check that catches that class of
error for every harness.

## Success Criteria

- A test, for each profile whose binary is present on PATH, runs the harness's
  `--help` (and any relevant subcommand `--help`) and asserts every flag and
  subcommand name the profile emits appears in that help output.
- When a profile's binary is absent, that profile's check is skipped with a
  logged notice naming the harness — it is not silently passed and not a hard
  failure (CI without the binary still goes green, but never reports the profile
  as verified).
- The test fails if a profile emits a flag or subcommand the installed binary
  does not define.
- The verification covers interactive, headless, and system-prompt argv for each
  profile, not just one mode.
