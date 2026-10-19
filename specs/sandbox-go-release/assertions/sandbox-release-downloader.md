---
id: sandbox-release-downloader
parent: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-go-release
---

# Sandbox Binary Is Downloaded from the Project's Public GitHub Release

`internal/sandbox/release.go` fetches the sandbox agent binary from a GitHub
Release on this project's own public repository. The cloud-init template is not
downloaded — it is embedded in the CLI binary — so no control-host
infrastructure files are bundled or fetched from a private repo.

Fetching is two-phase: the release metadata and cloud-init template are read up
front, but the agent binary is downloaded only once the sandbox machine's CPU
architecture is known, because which build to fetch depends on that machine.

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
