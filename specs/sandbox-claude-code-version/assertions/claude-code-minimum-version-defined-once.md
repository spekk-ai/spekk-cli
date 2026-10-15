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
- The same package reads a version from the output of `claude --version` (for example `2.1.301 (Claude Code)`), and tells whether a version is older than `MinVersion`. The comparison is numeric for each of the three parts: `2.1.1000` is newer than `2.1.292`, and `2.1.291` is older. Output with no `X.Y.Z` version at the start of a line is an error, not a version.
- The sandbox client and `spekk sandbox deploy` read `MinVersion` and the comparison from this package. No other Go file, template, or script holds the minimum version.
- Tests cover: a parse of real `--version` output, a parse failure, and the comparison at the boundary (`2.1.291`, `2.1.292`) and across a digit count (`2.1.1000`, `2.10.0`).
