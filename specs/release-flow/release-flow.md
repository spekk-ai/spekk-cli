---
id: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
---

# Release Flow

Cross-compile the spekk CLI for ARM and AMD on macOS, Linux, and Windows, publish versioned binaries as GitHub Release assets, and add a `spekk update` self-update command that authenticates via fine-grained PAT.

## Context

- 6 build targets: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`
- Repo is private — users authenticate with a fine-grained GitHub PAT scoped to `contents:read` on spekk-ai/spekk-cli
- GitHub Releases hosts the binaries (replaces Gemfury)
- The CLI updates itself in-place via `spekk update`

## Assertions

1. `version-embedding` — Binary embeds a version string via ldflags, exposed by `spekk --version`
2. `cross-compile-binaries` — Build produces 6 platform/arch binaries (depends on 1)
3. `github-release-publish` — CI uploads versioned binaries to GitHub Releases on tag push (depends on 2)
4. `self-update-command` — `spekk update` fetches latest release from GitHub API with `GH_SPEKK_TOKEN` (depends on 3)
5. `ci-release-pipeline` — CI workflow automates the full build → test → release pipeline (depends on 4)
