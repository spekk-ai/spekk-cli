---
id: claude-code-minimum-version-defined-once
parent: sandbox-claude-code-version
created: 2026-10-15T21:30:00Z
priority: 1
status: not_started
branch: fix/claude-code-version
---

# The Minimum Claude Code Version Is One Constant

## Success Criteria

- `internal/claudecode` holds `MinVersion = "2.1.292"`, with a comment that points to the measurements in the parent spec.
- The same package reads a version from the output of `claude --version`, and tells whether a version is older than `MinVersion`. The parse accepts `2.1.301 (Claude Code)`, `Claude Code v2.2.0`, and `claude 2.2.0`: a line that starts with `X.Y.Z`, or a line that contains `claude` (any case) and then `X.Y.Z` or `vX.Y.Z`. It skips other lines, such as an ssh warning that holds an IP address. Output with no such line is an error, not a version. The comparison is numeric for each of the three parts: `2.1.1000` is newer than `2.1.292`, and `2.1.291` is older.
- The sandbox client and `spekk sandbox deploy` read `MinVersion` and the comparison from this package. No other Go file, template, or script holds the minimum version.
- Tests cover: a parse of each of the three forms, of output after an ssh warning line with an IP address, a parse failure for text with no version, and the comparison at the boundary (`2.1.291`, `2.1.292`) and across a digit count (`2.1.1000`, `2.10.0`).

**Tests:** `internal/claudecode/version_test.go`
