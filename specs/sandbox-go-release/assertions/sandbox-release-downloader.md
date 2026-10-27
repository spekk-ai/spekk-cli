---
id: sandbox-release-downloader
parent: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-go-release
---

# Sandbox Binaries Are Downloaded from the Project's Public GitHub Release

`internal/sandbox/release.go` fetches the sandbox agent and spekk CLI binaries from a GitHub Release on this project's public repository. The cloud-init template is embedded in the CLI binary and is sent as in-memory data.

The CLI selects a release and pairs its concrete tag with the embedded cloud-init template. A pinned tag needs no release metadata request at this stage. After the machine's CPU architecture is known, the CLI reads the release digests and downloads the agent and CLI binaries for that architecture into one temp directory.

## Success Criteria

- `internal/sandbox/release.go` defines `fetchReleaseArtifacts(tag string)` which:
  - Resolves an empty or `"latest"` tag to the concrete tag of the newest
    release through the redirect of `github.com/{releaseRepo}/releases/latest`
    (`resolveReleaseTag`), and returns any other tag unchanged, where the
    `releaseRepo` constant is the public repo `spekk-ai/spekk-cli`
  - Sends no credential: the repository is public
  - Does NOT download any binary — it returns `*releaseArtifacts{Version,
    CloudInit}`, where `Version` is the concrete tag and `CloudInit` is the
    embedded template
- `releaseArtifacts` defines the method `downloadBinaries(arch string)` which:
  - Reads the sha256 digests of the release first, and fails before any download
    when an asset has none (`specs/sandbox-spekk-binaries/`,
    `release-binaries-sha256-checked`)
  - Downloads `sandbox-linux-{arch}` (via `sandboxAssetName`) and
    `spekk-linux-{arch}` (via `cliAssetName`) from
    `github.com/{releaseRepo}/releases/download/{tag}/{name}`
  - Writes them to a temp directory and records their paths in `BinaryPath` and
    `CLIPath`; `cleanup` removes the directory
  - Returns a clear error naming the asset and the release when that
    architecture's build is not present on the release
- The cloud-init template is provided by `//go:embed cloud-init.yaml`
  (`internal/sandbox/embed.go`), returned in the `CloudInit` field as in-memory
  bytes and sent straight to the DO API as droplet user-data — it is not a
  downloaded release asset.
- No private repository or application names appear in
  `internal/sandbox/release.go` — the only repository it references is the
  public one in its `releaseRepo` constant. (Verify with a case-insensitive
  search for the repository name this file referenced before this spec was
  reconciled; see the file's git history.)
