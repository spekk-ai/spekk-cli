---
id: ci-release-pipeline
parent: release-flow
created: 2026-06-04T12:00:00Z
priority: 1
status: done
depends-on: self-update-command
branch: temporary-target
---

# CI workflow automates the full release pipeline

When a version tag is pushed, GitHub Actions cross-compiles all 6 binaries with the version embedded via ldflags, and publishes them to Gemfury.

## Success Criteria

- Tag push (`v*`) triggers the release workflow in `.github/workflows/publish.yml`
- Workflow runs `go test ./...` before building
- Workflow uses `make build-all` to produce all 6 platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64, windows/arm64)
- Version is embedded via ldflags using the tag name (e.g., tag `v1.3.0` → binary reports `1.3.0`)
- Binaries are uploaded to Gemfury via the release script with `GEMFURY_TOKEN` from repository secrets
- Workflow fails clearly if any step fails (tests, build, publish)
- `workflow_dispatch` trigger is available for manual releases
