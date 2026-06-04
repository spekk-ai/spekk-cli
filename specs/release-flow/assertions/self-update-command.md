---
id: self-update-command
parent: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
status: done
depends-on: gemfury-publish
branch: temporary-target
---

# `spekk update` downloads and replaces the running binary

**Tests:** internal/update/update_test.go

The CLI can self-update by fetching the latest version from Gemfury and replacing itself in-place.

## Success Criteria

- `spekk update` checks Gemfury for the latest available version
- If a newer version exists, downloads the correct binary for the current OS/architecture
- Authentication uses HTTP basic auth header — token is never embedded in URLs (prevents leaking on redirects)
- Replaces the currently running binary in-place (handles file locking on Windows)
- Prints before/after version on success
- Skips update if already on latest version
- Requires `GEMFURY_TOKEN` environment variable for authentication
- Fails gracefully with clear error if token is missing, network is down, or permissions prevent replacement
- `spekk update --check` shows available version without installing
