---
id: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
---

# Release Flow

Cross-compile the spekk CLI for ARM and AMD on macOS, Linux, and Windows, publish versioned binaries as GitHub Release assets, and add a `spekk update` self-update command.

## Context

- 6 build targets: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`
- The repo is public. Release assets download with no credential, so `spekk update` needs no token
- GitHub Releases hosts the binaries (replaces Gemfury)
- The CLI updates itself in-place via `spekk update`

**History (recorded 2026-07-29):** this spec was written while spekk-ai/spekk-cli was private, and it required a fine-grained PAT in `GH_SPEKK_TOKEN`. The repository is public now, and `internal/update/` sends no `Authorization` header. The spec said the opposite of the code until this correction.

## Assertions

1. `version-embedding` — Binary embeds a version string via ldflags, exposed by `spekk --version`
2. `cross-compile-binaries` — Build produces 6 platform/arch binaries (depends on 1)
3. `github-release-publish` — CI uploads versioned binaries to GitHub Releases on tag push (depends on 2)
4. `self-update-command` — `spekk update` fetches latest release from the public GitHub API, unauthenticated (depends on 3)
5. `ci-release-pipeline` — CI workflow automates the full build → test → release pipeline (depends on 4)
