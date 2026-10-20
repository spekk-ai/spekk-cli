---
id: codex-auth-cross-build
parent: release-flow
created: 2026-10-20T16:05:00Z
priority: 1
status: done
branch: fix/rel142-windows-build
depends-on: cross-compile-binaries
---

# Codex auth checks permit release builds

## Success Criteria

- All six CLI targets and both agent targets in the release workflow build.
- Linux and macOS keep the private regular file, owner, and symlink checks before they read a Codex login cache. The check applies to the opened file to prevent a path swap during the check.
- A platform without those secure file checks reports a clear error when a caller supplies a Codex login cache.
- The exported sandbox behavior stays the same.

## Verification

- Run the credential tests and all CLI tests. Build each release target into a separate retained file. Run the Linux CLI version and help commands.
