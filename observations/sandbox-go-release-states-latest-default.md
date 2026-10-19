---
slug: sandbox-go-release-states-latest-default
type: outdated_specs
severity: medium
status: open
created: 2026-10-19T00:00:00Z
affected:
  - specs/sandbox-go-release/sandbox-go-release.md
  - specs/sandbox-go-release/assertions/release-tag-pins-the-source.md
  - specs/sandbox-go-release/assertions/sandbox-release-downloader.md
  - internal/sandbox/commands.go
  - internal/sandbox/release.go
---

# sandbox-go-release Still Declares "Latest Release" as the Default and the Old Download Path

## Issue Description

#288 (`sandbox-spekk-binaries`) changed which release a sandbox gets when no
`--release` is given: a CLI with a stable `vX.Y.Z` version now installs its own
release, and only a non-release build falls back to the latest release. It also
replaced the single agent download with `downloadBinaries`, which fetches both
the agent-client and the CLI and checks their sha256. The older
`sandbox-go-release` spec and two of its `done` assertions were not updated, so
they now contradict both the code and `default-release-is-the-operators`.

## Evidence

- `specs/sandbox-go-release/sandbox-go-release.md` line 19: "The release the
  artifacts come from is the latest published release by default".
- `specs/sandbox-go-release/assertions/release-tag-pins-the-source.md`: "By
  default spekk pulls ... from the latest published release", and the criterion
  "`releaseTag(pinned)` returns `"latest"` when `pinned` is empty". Its Note says
  `releaseTag` has no unit test.
- `internal/sandbox/commands.go` lines 106-114: `releaseTag("")` returns
  `version.Version` when it matches `releaseVersionRe`, and `"latest"` only
  otherwise. `internal/sandbox/release_test.go` (`TestReleaseTag`) now tests it.
- `specs/sandbox-go-release/assertions/sandbox-release-downloader.md` names
  `downloadAgentBinary(arch)`, a `releases/latest` API fetch with `GITHUB_TOKEN`
  (error when unset), and an asset-ID download through the API. In
  `internal/sandbox/release.go`, `downloadAgentBinary` is gone (removed in #288);
  `downloadBinaries` (line 124) reads digests, then `downloadAsset` (line 157)
  downloads both assets from `github.com/<repo>/releases/download/<tag>/<name>`
  with no token; `"latest"` resolves through the redirect in
  `resolveReleaseTag` (line 94). The stale comment on `fetchReleaseArtifacts`
  (line 75) also still names `downloadAgentBinary`.

## Impact

Two `done` specs disagree about the default release. A builder or reviewer
reading `release-tag-pins-the-source` would treat the new operator-release
default as a regression, and the downloader assertion describes a token
requirement that no longer exists, which misleads anyone checking whether a
sandbox create needs `GITHUB_TOKEN`.

## Recommendation

Accept the spec edit on this branch: state the operator-release default (pointing
to `sandbox-spekk-binaries/default-release-is-the-operators`), drop the
outdated "no unit test" note, and describe the download as it is now. The stale
code comment at `internal/sandbox/release.go` line 75 is for the builder.
