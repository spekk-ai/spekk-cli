---
id: cross-compile-binaries
parent: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
status: in_progress
depends-on: version-embedding
branch: temporary-target
locked-by: builder-Paris-MacBook-Pro-2.local-63231-1780528555
---

# Build produces 6 platform/arch binaries

A build script (or Makefile target) cross-compiles the spekk binary for all supported OS/architecture combinations.

## Success Criteria

- Build produces these 6 artifacts:
  - `spekk-darwin-amd64`
  - `spekk-darwin-arm64`
  - `spekk-linux-amd64`
  - `spekk-linux-arm64`
  - `spekk-windows-amd64.exe`
  - `spekk-windows-arm64.exe`
- Each binary embeds the version string (e.g., from git tag or `ldflags`)
- `spekk --version` prints the embedded version
- Build script is idempotent and can run from CI or locally
- All 6 binaries compile cleanly with `CGO_ENABLED=0` for static linking
