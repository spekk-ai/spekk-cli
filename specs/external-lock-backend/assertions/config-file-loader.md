---
id: config-file-loader
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 1
status: not_started
branch: feature/external-lock-backend
---

# Config File Loader

## What Must Be True

Spekk loads an optional `spekk.config.yaml` file from the repository root. The config exposes a `lock-backend` section that selects and configures which `LockStore` adapter to use. When no config file exists, spekk defaults to the `local` backend with current behavior.

## Success Criteria

- ✅ New package `internal/config/` with `config.go`
- ✅ `Config` struct with a `LockBackend` field
- ✅ `LockBackendConfig` struct supporting at minimum:
  ```yaml
  lock-backend:
    type: local | file | redis
    # type-specific fields:
    path: /path/to/locks            # for type: file
    url: redis://host:port          # for type: redis
    ttl: 2h                         # duration, optional, defaults to 2h
  ```
- ✅ `Load(repoRoot string) (*Config, error)` function that reads `spekk.config.yaml` from the repo root
- ✅ Missing config file is **not** an error — returns a default `Config{LockBackend: {Type: "local"}}`
- ✅ Invalid YAML or unknown `type` values return a clear error naming the file and field
- ✅ `ttl` field is parsed as a Go `time.Duration` via `time.ParseDuration`
- ✅ Unknown fields in the config file produce a warning but do not fail (forward-compat)
- ✅ Unit tests cover: missing file, valid local config, valid file config, valid redis config, invalid type, invalid ttl, invalid yaml
- ✅ Test fixtures placed in `internal/config/testdata/`

## Config Resolution Order

1. `./spekk.config.yaml` in the current working directory, walking up to the git root
2. If not found, defaults apply (`local` backend)

Environment variable overrides are **out of scope** for this assertion (can be added later if needed).

## Example Configs

**Redis:**
```yaml
lock-backend:
  type: redis
  url: redis://coord.team.internal:6379
  ttl: 2h
```

**File (shared mount):**
```yaml
lock-backend:
  type: file
  path: /Volumes/team-share/spekk-locks/
  ttl: 2h
```

**Explicit local:**
```yaml
lock-backend:
  type: local
```

## Out of Scope

- Constructing actual `LockStore` instances from the config (belongs with `wire-backend-into-next`)
- Environment variable overrides
- Config file location override via CLI flag (can be added later)
- Credentials for redis (for MVP, assume URL includes auth if needed, or unauthenticated)

## Notes

Keep the loader minimal. It reads, validates, and returns a struct. The wiring assertion is what turns this struct into running backends.
