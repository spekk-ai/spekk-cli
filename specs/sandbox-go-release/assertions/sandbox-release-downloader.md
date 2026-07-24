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

## Success Criteria

- `internal/sandbox/release.go` defines `fetchReleaseArtifacts(tag string)` which:
  - Fetches the release from `GET /repos/{releaseRepo}/releases/latest` when
    `tag` is empty or `"latest"`, otherwise `/releases/tags/{tag}`, where the
    `releaseRepo` constant is the public repo `spekk-ai/spekk-cli`
  - Uses `GITHUB_TOKEN` from env for auth (returns an error if it is unset)
  - Downloads one asset by name — the `sandbox-linux-amd64` binary — via the
    GitHub API asset endpoint (`/releases/assets/{id}` with
    `Accept: application/octet-stream`), following the 302 to the presigned
    objects host that carries its own auth
  - Writes the binary to a temp file and returns
    `*releaseArtifacts{Version, CloudInit, BinaryPath}`, where `Version` is the
    release tag name and `BinaryPath` is the temp file the caller removes
  - Returns a clear error including HTTP status and the repo when the release or
    the binary asset is not found
- The cloud-init template is provided by `//go:embed cloud-init.yaml`
  (`internal/sandbox/embed.go`), returned in the `CloudInit` field as in-memory
  bytes and sent straight to the DO API as droplet user-data — it is not a
  downloaded release asset.
- A case-insensitive search for `spekk-app` in `internal/sandbox/release.go`
  returns nothing.
