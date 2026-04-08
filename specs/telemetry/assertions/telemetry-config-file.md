---
id: telemetry-config-file
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
branch: feature/telemetry
---

# Telemetry Config File

## What Must Be True

Telemetry settings live in a user-global YAML file at `~/.config/spekk/telemetry.yaml` (respecting `XDG_CONFIG_HOME` when set). A Go package loads, validates, writes, and updates this file.

## Success Criteria

- ✅ New package `internal/telemetry/config/`
- ✅ `Config` struct maps to the following YAML shape:
  ```yaml
  enabled: true
  install_id: anon-9f3e8b2c4a1d...
  consented_at: 2026-04-08T17:00:00Z
  consent_version: 1
  endpoint: https://telemetry.spekk.ai/v1/events
  capture:
    coach_sessions: true
    spec_deltas: true
  redaction:
    extra_patterns:
      - "ACME_SECRET_.*"
  email: ""
  ```
- ✅ `Load() (*Config, error)` reads from `$XDG_CONFIG_HOME/spekk/telemetry.yaml`, falling back to `~/.config/spekk/telemetry.yaml`
- ✅ Missing file is **not** an error — returns `nil, nil` (telemetry disabled)
- ✅ Malformed YAML returns a clear error naming the file and failing field
- ✅ `Save(cfg *Config) error` writes the file with `0600` permissions (user-only read/write)
- ✅ `EnsureInstallID(cfg *Config)` generates a random `anon-{hex}` ID if missing and persists it
- ✅ `IsEnabled(cfg *Config) bool` returns true only if all of: config non-nil, `enabled: true`, `consented_at` present, `install_id` present
- ✅ Unit tests cover: missing file, valid full config, valid minimal config, malformed yaml, permissions check after save, install ID generation, `IsEnabled` truth table
- ✅ Test fixtures in `internal/telemetry/config/testdata/`

## XDG Compliance

On Linux and macOS, honor `$XDG_CONFIG_HOME` if set, else default to `$HOME/.config/spekk/telemetry.yaml`. Windows is out of scope for MVP.

## File Permissions

The config file must be written with mode `0600` because it contains the install ID (a stable identifier that could be correlated). The parent directory should be `0700`.

## Out of Scope

- Consent flow and `spekk telemetry enable` command (separate assertion)
- Repo-level override (separate assertion)
- Actual capture/upload logic (separate assertions)

## Notes

Keep this package narrow — it's just config I/O. No side effects beyond file read/write. This makes mocking telemetry in tests trivial: pass a `*Config` with `enabled: false` and nothing ever happens.
