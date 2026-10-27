---
slug: release-specs-predate-machine-download
type: outdated_specs
severity: medium
status: resolved
created: 2026-10-27T18:00:00Z
announced: 2026-10-27T19:03:41Z
affected:
  - specs/sandbox-go-release/assertions/sandbox-release-downloader.md
  - specs/sandbox-go-release/assertions/agent-binary-matches-machine-arch.md
  - specs/sandbox-spekk-binaries/assertions/release-binaries-sha256-checked.md
  - internal/sandbox/release.go
  - internal/sandbox/commands.go
  - internal/sandbox/release_test.go
---

# Release specs still say the CLI downloads the binaries on every deploy

## Issue Description

Commit 44c2b2d (#316, "Have the machine download the release binaries itself") split the old `downloadBinaries(arch string)` into two methods. `selectArch(arch)` reads the release digests and picks the builds, but downloads nothing. `downloadBinaries()` takes no argument and runs only in the scp fallback, when the machine cannot download the builds itself. The new assertion `machine-downloads-the-release` describes this. Two older `done` assertions in `sandbox-go-release` were not updated, and they still describe the old flow, where the CLI always downloads to a temp directory. A third assertion cites a test that was renamed.

## Evidence

- `specs/sandbox-go-release/assertions/sandbox-release-downloader.md`: the intro says that after the architecture is known, "the CLI reads the release digests and downloads the agent and CLI binaries for that architecture into one temp directory". The criteria define "`downloadBinaries(arch string)` which: Reads the sha256 digests of the release first … Downloads …". In `internal/sandbox/release.go` on main, `selectArch(arch string)` reads the digests and refuses an asset with no digest. `downloadBinaries()` takes no argument and returns early when `BinaryPath` is already set.
- `specs/sandbox-go-release/assertions/agent-binary-matches-machine-arch.md`: "`fetchAgentBinary` runs `detectArch` and then calls `downloadBinaries(arch)`." In `internal/sandbox/commands.go` (around line 346), `fetchAgentBinary` now ends with `return artifacts.selectArch(arch)`. The same assertion says "`downloadBinaries` removes its temp directory if a download fails", which still holds, but only on the fallback path.
- `specs/sandbox-spekk-binaries/assertions/release-binaries-sha256-checked.md`: its **Tests:** line cites `TestDownloadBinariesRefusesAnAssetWithNoDigest`. `internal/sandbox/release_test.go` has no test by that name. The test is now `TestSelectArchRefusesAnAssetWithNoDigest` (line 136). Its first criterion, "Before it downloads, the deploy reads the `digest`…", still holds.

## Impact

Someone who follows these assertions would expect a download to the operator's computer on every create and deploy, and an `arch` argument on `downloadBinaries`. A builder that re-runs either assertion, or a reviewer who checks it, could "fix" the code back toward the scp-only flow that #316 removed on purpose: an scp of about 40 MB breaks on networks that corrupt large transfers. The stale test name also makes the `Tests:` pointer impossible to follow.

## Recommendation

Update the two `sandbox-go-release` assertions to describe `selectArch(arch)` (digests and build choice, no download) and `downloadBinaries()` (the fallback download only), and point to `machine-downloads-the-release` for the download on the machine. Fix the test name in `release-binaries-sha256-checked`. The remedy commit on this branch makes these edits and changes no code.

Note: the open branch `observer/sandbox-go-release-states-latest-default` also edits `sandbox-release-downloader.md`, and its version of that criteria block already matches main. Whichever branch merges second may need a small conflict resolution in that block.