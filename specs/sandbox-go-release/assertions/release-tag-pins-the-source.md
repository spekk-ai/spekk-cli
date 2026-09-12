---
id: release-tag-pins-the-source
parent: sandbox-go-release
created: 2026-09-12T00:00:00Z
priority: 2
status: done
branch: sandbox-agent-arm64
---

# `--release <tag>` Pins the Release the Artifacts Come From

By default spekk pulls the cloud-init template and agent binary from the latest
published release. `--release <tag>` pins a specific one — the escape hatch for
an `exp-*` prerelease that carries a build the latest release does not (for
instance, an architecture published only in a prerelease).

## Success Criteria

- `spekk sandbox create`, `provision`, and `deploy` each accept `--release <tag>`.
- `releaseTag(pinned)` returns `"latest"` when `pinned` is empty and the tag
  otherwise, and every artifact fetch (`fetchArtifacts` / `fetchReleaseArtifacts`)
  goes through it — so no path hardcodes `"latest"` anymore.
- `CreateOptions.Release` and `ProvisionOptions.Release` carry the pinned tag;
  `Deploy(name, release string)` takes it as a parameter.
- The flag is documented for all three commands in `docs/cli-reference.md` and
  in each command's `--help` output.

**Note:** `releaseTag` and the flag plumbing have no unit test today. Behavior is
verifiable by hand (`--release exp-*` pulls from that tag) but a regression that
dropped the pin would not be caught by a test. A small test of `releaseTag("")`
→ `"latest"` and passthrough is worth adding.
