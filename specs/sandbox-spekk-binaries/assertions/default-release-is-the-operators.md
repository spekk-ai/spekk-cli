---
id: default-release-is-the-operators
parent: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
status: not_started
branch: feat/sandbox-spekk-binaries
---

# Without --release, a Sandbox Gets the Release of the Operator's CLI

## Success Criteria

- `spekk sandbox create`, `provision`, and `deploy` with no `--release`, from a CLI with a stable release version `vX.Y.Z`, install the binaries of `vX.Y.Z`.
- From any other build, they install the latest release, and print `This spekk is "<version>", not a release, so the sandbox gets the latest release, <tag>.` on stderr.
- `--release <tag>` still wins.
- The help text and `docs/cli-reference.md` say this, and no longer say "the latest spekk release".
