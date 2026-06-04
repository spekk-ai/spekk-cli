---
id: github-release-publish
parent: release-flow
created: 2026-06-04T12:00:00Z
priority: 1
status: not_started
depends-on: cross-compile-binaries
branch: temporary-target
---

# CI uploads versioned binaries to GitHub Releases on tag push

When a version tag is pushed, the release workflow builds all 6 platform binaries and attaches them as assets to a GitHub Release.

## Success Criteria

- Tag push (`v*`) triggers `.github/workflows/publish.yml`
- Workflow uses `make build-all` to produce all 6 binaries with version embedded via ldflags
- Binaries are attached to a GitHub Release created by `softprops/action-gh-release` (or equivalent)
- Asset names follow the pattern `spekk-{os}-{arch}` (e.g., `spekk-darwin-arm64`, `spekk-linux-amd64`)
- Release is tagged with the version (e.g., `v1.3.0`)
- `workflow_dispatch` trigger is available for manual releases
- Workflow fails clearly if build or upload fails
