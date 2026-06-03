---
id: absolute-paths-rejected-in-transcript-handler
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 1
status: done
branch: feature/spekk-sandbox-vulnrabilities
---

# Absolute paths are rejected in the transcript handler

The `BuildSkillMessage` function in `internal/agent/launcher.go` validates transcript file paths regardless of whether the input is relative or absolute. Absolute paths that point outside the working directory are rejected, closing the bypass where `filepath.IsAbs()` caused the traversal check to be skipped entirely.

## Success Criteria

- The path validation check applies to both relative and absolute paths — the `if !filepath.IsAbs(transcriptFile)` guard is removed or restructured so absolute paths are also validated
- `spekk coach meeting /etc/passwd` is rejected with a clear error message
- `spekk coach meeting ../../etc/passwd` is still rejected (existing behavior preserved)
- `spekk coach meeting notes.txt` (relative, within working directory) still works
- Absolute paths within the working directory (e.g., `/current/working/dir/notes.txt`) still work
- Tests cover absolute paths outside working directory, relative traversal paths, and legitimate paths

**Tests:** `internal/agent/launcher_test.go`
