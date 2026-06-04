---
id: self-update-command
parent: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
status: done
depends-on: github-release-publish
branch: temporary-target
---

# `spekk update` downloads and replaces the running binary

The CLI self-updates by fetching the latest release from the GitHub Releases API, authenticated with a fine-grained PAT.

## Success Criteria

- `spekk update` queries `api.github.com/repos/spekk-ai/spekk-cli/releases/latest` for the newest version
- Authentication uses `GITHUB_TOKEN` environment variable (fine-grained PAT with `contents:read`)
- Token is sent via `Authorization: token <PAT>` header — never embedded in URLs
- If a newer version exists, downloads the correct binary asset for the current OS/architecture
- Replaces the currently running binary in-place (handles file locking on Windows)
- Prints before/after version on success
- Skips update if already on latest version
- Requires `GITHUB_TOKEN` environment variable
- Fails gracefully with clear error if token is missing, network is down, or permissions prevent replacement
- `spekk update --check` shows available version without installing
- No Gemfury references remain in `internal/update/`
