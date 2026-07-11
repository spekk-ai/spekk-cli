---
id: refuses-dev-build
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 2
status: done
---

# Update Refuses to Run on a Development Build

## Description

A binary built from source without release ldflags reports
`version.Version == "dev"`. Self-updating such a build would silently replace
a developer's working binary with a release, so `spekk update` refuses
up front.

## Success Criteria

- When `version.Version` is `"dev"`, `spekk update` (and `--check`) returns
  the error "cannot update a development build; install a released version
  first" before making any network request
- Released binaries are built with `-ldflags "-X main.version=<tag>"`
  (publish.yml), so installed binaries report a real version and remain
  updatable

**Tests:** `TestRunDevBuild` in `internal/update/update_test.go`.
