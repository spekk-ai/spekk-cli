---
id: status-reports-binary-versions
parent: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
status: done
branch: feat/sandbox-spekk-binaries
depends-on: default-release-is-the-operators
---

# Sandbox Status Shows the Version of Each spekk Binary and Its Drift

## Success Criteria

- `spekk sandbox status <name>` prints `spekk CLI: <version>` and `Agent client: <version>`, read on the machine from `/opt/spekk/cli/spekk version` and `/opt/spekk/agent-client --version`, which run as `agent` (see `root-does-not-trust-agent-paths`).
- The expected version is the release that a `deploy` with no `--release` installs (see `default-release-is-the-operators`).
- A line gets ` WARNING: <reason>; update with: spekk sandbox deploy <name>` when the version is `dev` (no release stamp), not read (`missing`), or not the expected version. A version that matches has no warning. When the expected version cannot be found (no network), only `dev` and `missing` warn, and status says that it did not compare.
- Test: the classification of a match, a different version, `dev`, and a missing version.
- Live check: two containers, one with an old release CLI and one with a `dev` agent-client, each show the warning; after `spekk sandbox deploy`, both lines match with no warning.

**Tests:** `internal/sandbox/release_test.go` (`TestVersionDrift`). Live check (2026-10-18): after a create, `status` showed `spekk CLI: v1.38.1` and `Agent client: v1.38.1` with no warning. With the v1.32.0 CLI copied back on one box, it showed `spekk CLI: v1.32.0  WARNING: not v1.38.1, the release of this spekk; update with: spekk sandbox deploy box-a`. With a local `dev` agent-client on the other box, it showed `Agent client: dev  WARNING: a dev build, with no release stamp; ...`. After `spekk sandbox deploy` on each box, both lines matched with no warning.
