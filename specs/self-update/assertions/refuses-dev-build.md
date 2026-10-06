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
`version.Version == "dev"`. Self-updating such a build to "latest" would
silently replace a developer's working binary with a release, so a plain
`spekk update` refuses up front. Pinning `--version <tag>` is the deliberate
exception — the operator named a specific build — so the guard does not block it
(see `version-flag-installs-specific-tag`).

## Success Criteria

- When `version.Version` is `"dev"` and no `--version` is given, `spekk update`
  (and `--check`) returns an error containing "cannot update a development
  build" before making any network request
- The error points the operator at the escape hatch (install a released version
  first, or pin one with `--version`)
- `spekk update --version <tag>` is allowed from a dev build (it bypasses this
  guard)
- Released binaries are built with `-ldflags "-X main.version=<tag>"`
  (publish.yml), so installed binaries report a real version and remain
  updatable

**Tests:** `TestRunDevBuild` and `TestRunVersionBypassesDevGuardAndFetchesByTag`
in `internal/update/update_test.go`.
