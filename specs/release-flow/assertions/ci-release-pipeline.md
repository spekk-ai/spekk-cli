---
id: ci-release-pipeline
parent: release-flow
created: 2026-06-04T12:00:00Z
priority: 1
status: done
depends-on: self-update-command
branch: temporary-target
---

# CI workflow automates the full build, test, and release pipeline

When a version tag is pushed, GitHub Actions runs tests, cross-compiles all 6 binaries with the version embedded, and publishes them as GitHub Release assets.

## Success Criteria

- Tag push (`v*`) triggers the release workflow in `.github/workflows/publish.yml`
- Workflow runs `go test ./...` before building
- Workflow uses `make build-all` to produce all 6 platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64, windows/arm64)
- Version is embedded via ldflags using the tag name (e.g., tag `v1.3.0` → binary reports `1.3.0`)
- Binaries are uploaded as GitHub Release assets via `softprops/action-gh-release` (or equivalent)
- No Gemfury references remain in CI workflows or release scripts
- Workflow fails clearly if any step fails (tests, build, release)
- `workflow_dispatch` trigger is available for manual releases
