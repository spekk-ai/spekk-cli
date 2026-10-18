---
id: status-reports-binary-versions
parent: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
status: not_started
branch: feat/sandbox-spekk-binaries
depends-on: default-release-is-the-operators
---

# Sandbox Status Shows the Version of Each spekk Binary and Its Drift

## Success Criteria

- `spekk sandbox status <name>` prints `spekk CLI: <version>` and `Agent client: <version>`, read on the machine from `/opt/spekk/cli/spekk version` and `/opt/spekk/agent-client --version`.
- The expected version is the release that a `deploy` with no `--release` installs (see `default-release-is-the-operators`).
- A line gets ` WARNING: <reason>; update with: spekk sandbox deploy <name>` when the version is `dev` (no release stamp), not read (`missing`), or not the expected version. A version that matches has no warning. When the expected version cannot be found (no network), only `dev` and `missing` warn, and status says that it did not compare.
- Test: the classification of a match, a different version, `dev`, and a missing version.
- Live check: two containers, one with an old release CLI and one with a `dev` agent-client, each show the warning; after `spekk sandbox deploy`, both lines match with no warning.
