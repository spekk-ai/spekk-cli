---
id: version-flag-installs-specific-tag
parent: self-update
created: 2026-09-12T00:00:00Z
priority: 2
status: done
branch: sandbox-agent-arm64
---

# `spekk update --version <tag>` Installs a Specific Release

Without `--version`, `spekk update` installs whatever is newest, with guards. But
GitHub excludes prereleases from its "latest" endpoint, so an `exp-*` build is
unreachable that way. `--version <tag>` names the exact release to install,
which is how an operator gets onto (or back off) a prerelease.

## Success Criteria

- `spekk update --version <tag>` fetches the release by tag via `FetchRelease`,
  which hits `/repos/{owner}/{repo}/releases/tags/{tag}` — not `/releases/latest`.
- A pinned tag installs exactly as named, skipping both the newer-than-current
  check and the development-build guard, because the operator asked for that
  specific build rather than "whatever is newest".
- `--version` therefore works from a development build (`spekk version` prints
  `dev`), where a plain `spekk update` is refused.
- `spekk update --check --version <tag>` previews only: it prints the current
  version and what it would install, writes nothing, and downloads no asset.
- The flag is documented in `docs/cli-reference.md` and in `spekk update --help`.

**Tests:** `TestRunVersionBypassesDevGuardAndFetchesByTag` in
`internal/update/update_test.go` (verifies the dev guard is bypassed and the
fetch hits `/releases/tags/<tag>`).
