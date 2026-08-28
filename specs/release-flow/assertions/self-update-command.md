---
id: self-update-command
parent: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
status: done
depends-on: github-release-publish
---

# `spekk update` downloads and replaces the running binary

The CLI self-updates by fetching the latest release from the GitHub Releases API. The repository is public, so the request carries no credential.

## Success Criteria

- `spekk update` queries `api.github.com/repos/spekk-ai/spekk-cli/releases/latest` for the newest version
- The request is unauthenticated. It sends no `Authorization` header and reads no token from the environment
- If a newer version exists, downloads the correct binary asset for the current OS/architecture
- Replaces the currently running binary in-place (handles file locking on Windows)
- Prints before/after version on success
- Skips update if already on latest version
- Fails gracefully with clear error if the network is down, GitHub returns a non-200 status, or permissions prevent replacement
- `spekk update --check` shows available version without installing
- No Gemfury references remain in `internal/update/`

**Known constraint:** unauthenticated GitHub API calls have a per-IP rate limit (60 per hour). A host that updates many times in one hour, or many sandboxes behind one address, can receive a 403. The error message shows the GitHub status and body, so the cause is legible.

**Correction (recorded, 2026-07-29):** this assertion required a fine-grained PAT in `GH_SPEKK_TOKEN`, sent as an `Authorization: token <PAT>` header. That was correct while the repository was private. The repository is public now, and `internal/update/update.go` sends no such header — `FetchLatestRelease` sets only the `Accept` header. The criteria above now match the code. A future move back to a private repository must change the code and this assertion together.
