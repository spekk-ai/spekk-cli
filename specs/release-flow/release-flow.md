---
id: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
---

# Release Flow

Cross-compile the spekk CLI for ARM and AMD on macOS, Linux, and Windows, publish versioned artifacts to Gemfury for token-gated distribution, and add a `spekk update` self-update command.

## Context

- 6 build targets: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`
- Gemfury provides private package hosting — only users with a valid token can download updates
- The CLI should be able to update itself in-place via `spekk update`

## Assertions

1. `version-embedding` — Binary embeds a version string via ldflags, exposed by `spekk --version`
2. `cross-compile-binaries` — Build produces 6 platform/arch binaries (depends on 1)
3. `gemfury-publish` — Release script uploads versioned artifacts to Gemfury (depends on 2)
4. `self-update-command` — `spekk update` downloads and replaces the running binary (depends on 3)
5. `ci-release-pipeline` — CI workflow cross-compiles, embeds version, and publishes to Gemfury on tag push (depends on 4)
