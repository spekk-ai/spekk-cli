---
id: harness-profile-abstraction
parent: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 2
status: done
branch: feature/configurable-harness
---

# Launch sites resolve the harness through a profile, not a `"claude"` literal

Every place spekk spawns an interactive or headless agent reads its binary name
and flags from a harness profile value. `serve` is excluded (see
`serve-stays-claude-only`).

**Tests:** internal/agent/harness_test.go

## Success Criteria

- A harness profile type carries: the binary name, how a prompt argument is
  passed, the flag(s) for skipping permissions, the flag(s) for headless mode,
  and the "binary not found — install X" guidance text.
- No launch site in `internal/agent/` contains the string literal `"claude"` as
  a binary name; each obtains its binary from the resolved profile. (`serve` is
  out of scope and unchanged.)
- The interactive launch (`launcher.go`), the builder launch (`builder.go`), and
  the headless launch (`launcher_headless_unix.go` / `_windows.go`) all spawn the
  profile's binary with the profile's flags.
- `observer_cron.go` looks up and bakes the profile's binary into the crontab
  entry (not a hardcoded `claude`).
- When the resolved binary is missing, the error names the profile's harness and
  its install guidance — not a hardcoded "install Claude Code" for a non-claude
  harness.
- With no `--harness` flag and no `SPEKK_HARNESS` set, behaviour is byte-for-byte
  identical to today: the `claude-code` profile spawns `claude` with the same
  flags as before. A test asserts the resolved argv for the default profile.
