---
id: sandbox-release-downloader
parent: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-go-release
---

# Sandbox Artifacts Are Downloaded from spekk-app GitHub Release

`src/sandbox/release.js` fetches the sandbox binary and cloud-init template from a spekk-app GitHub Release. No infrastructure files are bundled in this repo.

## Success Criteria

- `src/sandbox/release.js` exports `fetchReleaseArtifacts(tag = 'latest')` which:
  - Fetches the release from `GET /repos/spekk-ai/spekk-app/releases/tags/{tag}` (or `/releases/latest` when tag is `'latest'`)
  - Uses `GITHUB_TOKEN` from env for auth (already required for other sandbox commands)
  - Downloads two assets by name: `sandbox` (binary) and `cloud-init.yaml` using the GitHub API asset endpoint (`/releases/assets/{id}` with `Accept: application/octet-stream`) — not `browser_download_url`, which drops auth on redirect for private repos
  - Writes both to temp files and returns `{ binaryPath, cloudInitPath, version }` where `version` is the release tag name
  - Prints a clear error with HTTP status and exits if the release or either asset is not found
- `src/sandbox/templates/cloud-init.yaml` does not exist — it is no longer bundled
- `src/sandbox/templates.js` does not export `fetchAgentClient()` — that function is removed

**Tests:** src/sandbox/__tests__/release.test.js
