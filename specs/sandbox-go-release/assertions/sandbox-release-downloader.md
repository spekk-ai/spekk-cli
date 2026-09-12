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
  - Fetches the release from `GET /repos/{releaseRepo}/releases/latest` when
    `tag` is empty or `"latest"`, otherwise `/releases/tags/{tag}`, where the
    `releaseRepo` constant is the public repo `spekk-ai/spekk-cli`
  - Uses `GITHUB_TOKEN` from env for auth (returns an error if it is unset)
  - Does NOT download the agent binary — it returns
    `*releaseArtifacts{Version, CloudInit, release, token}`, where `Version` is
    the release tag name, `CloudInit` is the embedded template, and `release` +
    `token` are retained so the binary can be fetched later
  - Returns a clear error including HTTP status and the repo when the release is
    not found
- `releaseArtifacts` defines the method `downloadAgentBinary(arch string)` which:
  - Downloads the asset named `sandbox-linux-{arch}` (via `sandboxAssetName`) by
    ID through the GitHub API asset endpoint (`/releases/assets/{id}` with
    `Accept: application/octet-stream`), following the 302 to the presigned
    objects host that carries its own auth
  - Writes the binary to a temp file and records its path in `BinaryPath`, which
    the caller removes when done
  - Returns a clear error naming the asset when that architecture's build is not
    present on the release
- The cloud-init template is provided by `//go:embed cloud-init.yaml`
  (`internal/sandbox/embed.go`), returned in the `CloudInit` field as in-memory
  bytes and sent straight to the DO API as droplet user-data — it is not a
  downloaded release asset.
- No private repository or application names appear in
  `internal/sandbox/release.go` — the only repository it references is the
  public one in its `releaseRepo` constant. (Verify with a case-insensitive
  search for the repository name this file referenced before this spec was
  reconciled; see the file's git history.)
