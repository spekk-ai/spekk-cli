---
id: release-tag-pins-the-source
parent: sandbox-go-release
created: 2026-09-12T00:00:00Z
priority: 2
status: done
branch: sandbox-agent-arm64
---

# `--release <tag>` Pins the Release the Artifacts Come From

With no `--release`, spekk pulls the binaries from the release of the operator's
CLI when it is a stable `vX.Y.Z` build, and from the latest published release
otherwise (`specs/sandbox-spekk-binaries/assertions/default-release-is-the-operators.md`).
`--release <tag>` pins a specific one — the escape hatch for
an `exp-*` prerelease that carries a build the latest release does not (for
instance, an architecture published only in a prerelease).

## Success Criteria

- `spekk sandbox create`, `provision`, and `deploy` each accept `--release <tag>`.
- `releaseTag(pinned)` returns `pinned` when it is set; else the CLI's own
  version when it matches `vX.Y.Z`; else `"latest"`. Every artifact fetch (`fetchArtifacts` / `fetchReleaseArtifacts`)
  goes through it — so no path hardcodes `"latest"` anymore.
- `CreateOptions.Release` and `ProvisionOptions.Release` carry the pinned tag;
  `Deploy(name, release string)` takes it as a parameter.
- The flag is documented for all three commands in `docs/cli-reference.md` and
  in each command's `--help` output.

**Tests:** `internal/sandbox/release_test.go` (`TestReleaseTag`).
