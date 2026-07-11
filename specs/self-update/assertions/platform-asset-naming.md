---
id: platform-asset-naming
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 2
status: done
---

# Update Selects the Correct Platform Asset

## Description

The updater downloads the release asset matching the running platform, using
the same naming scheme the release pipeline and `install.sh` use.

## Success Criteria

- Expected asset name is `spekk-{goos}-{goarch}` (e.g. `spekk-darwin-arm64`,
  `spekk-linux-amd64`), computed by `AssetName(runtime.GOOS, runtime.GOARCH)`
- On Windows the name carries an `.exe` suffix (`spekk-windows-amd64.exe`)
- The asset is matched by exact name against the release's asset list and
  downloaded via its `browser_download_url` with
  `Accept: application/octet-stream`
- If the release has no asset for the current platform, the error names the
  platform and the release tag
  ("no binary found for <goos>/<goarch> in release <tag>")

**Tests:** `TestAssetName` in `internal/update/update_test.go`.
