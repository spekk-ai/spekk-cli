---
id: checks-github-releases-api
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 1
status: done
---

# Update Checks the GitHub Releases API Without Authentication

## Description

`spekk update` discovers the latest version by querying the public GitHub
Releases API. No token, login, or other credentials are required — the same
zero-auth posture as `install.sh`.

## Success Criteria

- Latest release fetched from
  `https://api.github.com/repos/spekk-ai/spekk-cli/releases/latest` with
  `Accept: application/vnd.github+json`
- No `Authorization` header is sent on the API request or the asset download
- The latest version is read from the release `tag_name` with any leading `v`
  stripped; an empty tag yields a "no releases found" error
- Network errors and non-200 API responses surface as clear errors
  (status code and response body included), wrapped as
  "failed to check for updates"
- HTTP client is injectable (`update.Client`) so tests run without real
  network access

**Tests:** `TestFetchLatestRelease`, `TestFetchLatestReleaseAPIError` in
`internal/update/update_test.go` (both verify the absence of an
`Authorization` header via an injected transport).
