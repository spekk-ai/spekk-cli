---
id: self-update
created: 2026-06-12T00:00:00Z
priority: 2
---

# Self-Update Command

`spekk update` upgrades the installed binary in place from the latest GitHub
release, with no package manager and no auth token required. `spekk update
--check` previews what would happen without touching the filesystem.

```bash
spekk update          # download and install the latest release
spekk update --check  # show current vs. latest version, install nothing
```

Shipped in v1.5.0 (PR #107); implementation lives in `internal/update/`
(`update.go`, tests in `update_test.go`). PR #114 hardened the permission
failure mode (see `fail-fast-permission-error`).

## Design

- **Release source:** `https://api.github.com/repos/spekk-ai/spekk-cli/releases/latest`,
  unauthenticated. Release assets are named `spekk-{goos}-{goarch}`
  (plus `.exe` on Windows), matching what `publish.yml` uploads and what
  `install.sh` downloads.
- **In-place atomic replace:** the new binary is written to a temp file in the
  same directory as the running binary (resolved through symlinks via
  `os.Executable` + `filepath.EvalSymlinks`), then renamed over it. Same-dir
  temp + rename keeps the swap atomic on POSIX filesystems.
- **Fail fast on permissions:** the temp file is created *before* any download,
  so a read-only install dir (e.g. a sudo-installed `/usr/local/bin`) fails
  immediately with an actionable message instead of after wasting a download.
- **No auto-sudo:** the binary never escalates privileges itself; it tells the
  user to re-run with sudo. The companion fix is moving the default install
  location to a user-owned directory (see the `install-script` spec).

## Assertions

See `assertions/` for what must be true about the update flow.
